package curtprocessing

import (
   "fmt"
   "http"
   "io/ioutil"
   "json"
   "strconv"
   "strings"
   "appengine"
   "appengine/urlfetch"
)

func init(){
   http.HandleFunc("/",get_id)
}

func get_id(w http.ResponseWriter, r *http.Request){
  fmt.Fprint(w,"Beginning to process images...")
  c := appengine.NewContext(r)
  client := urlfetch.Client(c)
  
  api_resp, err := client.Get("http://docs.curthitch.biz/api/getallpartid?dataType=json")
  
  if err != nil{
    fmt.Fprint(w,err)
  }else{
      var b []byte;
      if err == nil{
         b, err = ioutil.ReadAll(api_resp.Body);
         api_resp.Body.Close();
         if b != nil{
            fmt.Fprint(w, "We retrieved the ids");
         }else{
            fmt.Fprint(w, "we did not retrieve the ids");
         }
      }

      if err != nil {
         fmt.Println(err)
            fmt.Println("Whoa we really didn't make it!")
      }else{
         var ids []float64;
         err = json.Unmarshal(b, &ids)
         if err != nil{
            fmt.Println(err)
         }else{

            // Loop through the ids
            for _, value := range ids{

                // Loop through the indexes
                keys := [...]string{"a","b","c","d","e","p","q"}
                for _,char := range keys{

                    // Loop the sizes
                    sizes := [...]string{"100x75","200x238","300x225","1024x768","3008x1990"}
                    for _,size := range sizes{
                        var str string;
                        str = strconv.Ftoa64(value,'f',0)
                        var url string;
                        url = "http://docs.curthitch.biz/masterlibrary/"+str+"/images/"+str+"_"+size+"_"+char+".jpg"
                        resp, err := client.Head(url)
                        if err != nil {
                          fmt.Fprint(w,err.String()+"\n");
                          err = nil;
                        }else{
                            if(resp.StatusCode == 404){
                                fmt.Fprint(w,url+"\r\n")
                                fmt.Fprint(w,"---------------------------------------------\r\n")
                                fmt.Fprint(w,"not found")
                                fmt.Fprint(w,"\r\n\n")
                            }else{
                                dims := strings.Split(size,"x")
                                api_url := ""
                                if(dims[0] == "100"){
                                    api_url = "http://docs.curthitch.biz/api/AddImage?partID="+str+"&sort="+char+"&path="+url+"&height="+dims[1]+"&width="+dims[0] +"&size=Tall"
                                }else if(dims[0] == "200"){
                                    api_url = "http://docs.curthitch.biz/api/AddImage?partID="+str+"&sort="+char+"&path="+url+"&height="+dims[1]+"&width="+dims[0] +"&size=Medio"
                                }else if(dims[0] == "300"){
                                    api_url = "http://docs.curthitch.biz/api/AddImage?partID="+str+"&sort="+char+"&path="+url+"&height="+dims[1]+"&width="+dims[0] +"&size=Grande"
                                }else if(dims[0] == "1024"){
                                    api_url = "http://docs.curthitch.biz/api/AddImage?partID="+str+"&sort="+char+"&path="+url+"&height="+dims[1]+"&width="+dims[0] +"&size=Venti"
                                }else if(dims[0] == "3008"){
                                    api_url = "http://docs.curthitch.biz/api/AddImage?partID="+str+"&sort="+char+"&path="+url+"&height="+dims[1]+"&width="+dims[0] +"&size=Trenta"
                                }
                                if(api_url != ""){
                                    post_resp, api_err := client.Get(api_url)
                                    if api_err != nil{
                                        fmt.Fprint(w,err.String()+"\r\n")
                                    }else{
                                        if post_resp.StatusCode != 200{
                                            fmt.Fprint(w,"Publish failed for "+api_url+"\r\n")
                                        }
                                    }
                                    if post_resp != nil{
                                        post_resp.Body.Close()
                                    }
                                }
                            } // End if resp.StatusCode

                         } // End if err != nil

                         if resp != nil{
                            resp.Body.Close()
                         }

                    }// End for sizes

                } // End for keys

            }
         }
      }
  }
}
