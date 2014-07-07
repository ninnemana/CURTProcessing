package main

import (
	"bytes"
	"code.google.com/p/goauth2/oauth"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ninnemana/google-api-go-client/storage/v1"
	"net/http"
)

var (
	getPartImages = `select
										pi.imageID, pi.sort, pi.path, pi.height, pi.width, pi.partID,
										pis.size, pis.dimensions
										from PartImages pi
										join PartImageSizes as pis on pi.sizeID = pis.sizeID
										order by pi.partID`

	config = oauth.Config{}
)

type Part struct {
	Images []PartImage
	ID     int
}

type PartImage struct {
	ID         int
	Size       string
	Path       string
	Height     int
	Width      int
	PartID     int
	Sort       string
	Dimensions string
}

func main() {

	config.AuthURL = "https://accounts.google.com/o/oauth2/auth"
	config.TokenURL = "https://accounts.google.com/o/oauth2/token"
	config.ClientId = "437356853190-aj2vckticb9kormhboma5f1ntp13lspi.apps.googleusercontent.com"
	config.ClientSecret = "i8fCArIKVqEb2Gjzfh_0qmJf"
	config.RedirectURL = "http://localhost:8000/oauth2callback"
	config.Scope = storage.DevstorageFull_controlScope

	http.HandleFunc("/", index)
	http.HandleFunc("/oauth2callback", callback)

	http.ListenAndServe(":8000", nil)
}

func index(rw http.ResponseWriter, r *http.Request) {

	url := config.AuthCodeURL(r.URL.Path)
	http.Redirect(rw, r, url, http.StatusFound)
}

func callback(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	code := params.Get("code")
	t := &oauth.Transport{Config: &config}

	if t == nil {
		http.Redirect(rw, r, "/", http.StatusFound)
		return
	}

	token, err := t.Exchange(code)
	if err != nil {
		http.Redirect(rw, r, "/", http.StatusFound)
		return
	}
	t.Token = token

	client := t.Client()

	serv, err := storage.New(client)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	buckets, err := serv.Buckets.List("responsive-seat-251").Do()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var partBucket *storage.Bucket
	for _, buck := range buckets.Items {
		if buck.Name == "curt-parts" {
			partBucket = buck
		}
	}

	if partBucket == nil {
		partBucket = &storage.Bucket{
			Name: "curt-parts",
		}
		partBucket, err = serv.Buckets.Insert("responsive-seat-251", partBucket).Do()
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	parts, err := getAllImages()
	if err != nil {
		panic(err)
	}

	for _, p := range parts {
		// setup directory structure for this part
		if err := p.setupDirs(serv); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Part #%d has (%d) images\n", p.ID, len(p.Images))
		for _, i := range p.Images {
			go i.getImage(serv)
		}
		fmt.Fprintln(rw, fmt.Sprintf("Processed part #%s", p.ID))
	}
}

func (p *Part) setupDirs(serv *storage.Service) error {
	r := bytes.NewReader([]byte(""))

	// Get the base folder
	folder := fmt.Sprintf("%d/", p.ID)
	obj, err := serv.Objects.Get("curt-parts", folder).Do()
	if obj == nil || err != nil { // create
		obj = &storage.Object{
			Name: folder,
		}
		obj, err = serv.Objects.Insert("curt-parts", obj).Media(r).Do()
		if err != nil {
			return err
		}
	}

	// get the images folder
	folder = fmt.Sprintf("%d/images/", p.ID)
	obj, err = serv.Objects.Get("curt-parts", folder).Do()
	if obj == nil || err != nil {
		obj = &storage.Object{
			Name: folder,
		}
		obj, err = serv.Objects.Insert("curt-parts", obj).Media(r).Do()
		if err != nil {
			return err
		}
	}

	// get the installation folder
	folder = fmt.Sprintf("%d/install/", p.ID)
	obj, err = serv.Objects.Get("curt-parts", folder).Do()
	if obj == nil || err != nil {
		obj = &storage.Object{
			Name: folder,
		}
		obj, err = serv.Objects.Insert("curt-parts", obj).Media(r).Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *PartImage) getImage(serv *storage.Service) {
	r := bytes.NewReader([]byte(""))
	resp, err := http.Get(i.Path)
	if err != nil {
		// log error
		fmt.Printf("failed to get image %s: %s\n", i.Path, err.Error())
		return
	}
	defer resp.Body.Close()

	// check if there is a folder for this size
	folder := fmt.Sprintf("%d/images/%s/", i.PartID, i.Size)
	obj, err := serv.Objects.Get("curt-parts", folder).Do()
	if obj == nil {
		obj = &storage.Object{
			Name: folder,
		}
		if _, err = serv.Objects.Insert("curt-parts", obj).Media(r).Do(); err != nil {
			// log error...
			fmt.Printf("failed to load image %s: %s\n", i.Path, err.Error())
			return
		}
	}

	// check if this object already exists
	img := fmt.Sprintf("%d/images/%s/%d_%s.jpg", i.PartID, i.Size, i.PartID, i.Sort)
	obj, err = serv.Objects.Get("curt-parts", img).Do()
	if obj == nil {
		obj = &storage.Object{
			Name: img,
		}
	}
	acl := storage.ObjectAccessControl{}
	obj.Acl = append(obj.Acl, acl)
	obj, err = serv.Objects.Insert("curt-parts", obj).Media(resp.Body).Do()

	if err != nil {
		// log error
		fmt.Printf("failed to load image %s: %s\n", i.Path, err.Error())
	}

	fmt.Printf("Image %s for size %s processed under part #%d\n", i.Sort, i.Size, i.PartID)
}

func getAllImages() (map[int]Part, error) {

	indexParts := make(map[int]Part, 0)

	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/CurtDev?parseTime=true&loc=America%2FChicago")
	if err != nil {
		return indexParts, err
	}

	query, err := db.Prepare(getPartImages)
	if err != nil {
		return indexParts, err
	}

	rows, err := query.Query()
	if err != nil {
		return indexParts, err
	}

	for rows.Next() {
		var img PartImage
		if err := rows.Scan(&img.ID, &img.Sort, &img.Path, &img.Height, &img.Width, &img.PartID, &img.Size, &img.Dimensions); err != nil {
			panic(err)
		}

		if part, ok := indexParts[img.PartID]; !ok {
			indexParts[img.PartID] = Part{
				ID:     img.PartID,
				Images: []PartImage{img},
			}
		} else {
			part.Images = append(part.Images, img)
			indexParts[img.PartID] = part
		}
	}

	return indexParts, nil
}
