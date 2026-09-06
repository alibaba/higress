// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0
package main

import (
 "bufio"
 "bytes"
 "context"
 "crypto/sha256"
 "crypto/tls"
 "encoding/json"
 "flag"
 "fmt"
 "io"
 "net"
 "net/http"
 "os"
 "strings"
 "time"
)

type obj = map[string]interface{}
func main() {
 node:=flag.String("node", "", "kind node IP")
 variant:=flag.String("variant", "fixed", "baseline or fixed")
 flag.Parse()
 tr:=&http.Transport{ForceAttemptHTTP2:true,TLSClientConfig:&tls.Config{InsecureSkipVerify:true},DialContext:func(ctx context.Context,network,address string)(net.Conn,error){_,port,_:=net.SplitHostPort(address);return (&net.Dialer{}).DialContext(ctx,network,net.JoinHostPort(*node,port))}}
 client:=&http.Client{Transport:tr,Timeout:5*time.Second}
 failures:=0
 check:=func(name string,ok bool,detail interface{}) { json.NewEncoder(os.Stdout).Encode(obj{"case":name,"pass":ok,"detail":detail});if !ok {failures++} }
 call:=func(host,path,method string,params obj,trailers bool)(*http.Response,[]byte,error){
  verb:="GET";var body io.Reader
  if method!="" {verb="POST";b,_:=json.Marshal(obj{"jsonrpc":"2.0","id":"verify-1","method":method,"params":params});body=bytes.NewReader(b)}
  req,_:=http.NewRequest(verb,"https://"+host+":30443"+path,body)
  req.Header.Set("Content-Type","application/json");req.Header.Set("A2A-Version","0.3");req.Header.Set("X-Higress-A2A-Method","spoofed")
  if trailers {req.ContentLength=-1;req.Trailer=http.Header{"X-Demo-Request-Trailer":[]string{"complete"}}}
  resp,err:=client.Do(req);if err!=nil{return nil,nil,err};defer resp.Body.Close();b,err:=io.ReadAll(resp.Body);return resp,b,err
 }
 for _,source:=range []string{"k8s","remote"} {
  host:=source+".agents.test";expected:="kubernetes-agent";if source=="remote" {expected="remote-agent"}
  for _,path:=range []string{"/.well-known/agent-card.json","/.well-known/agent.json","/.well-known/agent-card.json?version=1.0"} {
   resp,b,err:=call(host,path,"",nil,false)
   check(source+" card "+path,err==nil&&resp.StatusCode==200&&bytes.Contains(b,[]byte("https://"+host+":30443"))&&bytes.Contains(b,[]byte(expected)),string(b))
  }
  params:=obj{"message":obj{"messageId":"m1","role":"user","parts":[]obj{{"kind":"text","text":"hello"}}}}
  resp,b,err:=call(host,"/?working=true","message/send",params,false)
  check(source+" send",err==nil&&resp.StatusCode==200&&resp.Header.Get("X-Higress-A2A-Task-State")=="working"&&resp.Header.Get("X-Demo-Agent")==expected,string(b))
  var envelope struct{Result struct{ID string `json:"id"`} `json:"result"`};json.Unmarshal(b,&envelope)
  for _,method:=range []string{"tasks/get","tasks/cancel"} {
   resp,b,err=call(host,"/",method,obj{"id":envelope.Result.ID},false)
   state:="working";if method=="tasks/cancel"{state="canceled"}
   check(source+" "+method,err==nil&&resp.StatusCode==200&&resp.Header.Get("X-Higress-A2A-Task-State")==state,string(b))
  }
  resp,b,err=call(host,"/","unknown",obj{},false)
  check(source+" reject unknown",err==nil&&resp.StatusCode==400&&bytes.Contains(b,[]byte("-32601")),string(b))
  // Streaming must deliver its first event before the Agent's 600ms delay.
  data,_:=json.Marshal(obj{"jsonrpc":"2.0","id":"stream","method":"message/stream","params":params})
  req,_:=http.NewRequest("POST","https://"+host+":30443/",bytes.NewReader(data));req.Header.Set("Content-Type","application/json")
  start:=time.Now();sr,se:=client.Do(req)
  if se!=nil {check(source+" stream",false,se.Error())} else {
   scanner:=bufio.NewScanner(sr.Body);count:=0;first:=time.Duration(0);var events []string
   for scanner.Scan(){line:=scanner.Text();if strings.HasPrefix(line,"data:"){if count==0{first=time.Since(start)};count++;events=append(events,line)}}
   sr.Body.Close();check(source+" stream",scanner.Err()==nil&&count==2&&first<500*time.Millisecond&&time.Since(start)>=600*time.Millisecond,obj{"first_ms":first.Milliseconds(),"total_ms":time.Since(start).Milliseconds(),"events":events})
  }
  for i:=0;i<3;i++ {
   resp,b,err=call(host,"/.well-known/agent-card.json?trailers=true","",nil,false)
   rewritten:=err==nil&&resp.StatusCode==200&&bytes.Contains(b,[]byte("https://"+host+":30443"))
   check(fmt.Sprintf("%s trailers card %d",source,i),rewritten==(*variant=="fixed"),obj{"rewritten":rewritten,"body":string(b),"sha256":fmt.Sprintf("%x",sha256.Sum256(b)),"error":fmt.Sprint(err)})
   resp,b,err=call(host,"/?trailers=true","message/send",params,false)
   metadata:=err==nil&&resp.Header.Get("X-Higress-A2A-Task-State")=="completed"
   check(fmt.Sprintf("%s trailers unary %d",source,i),metadata==(*variant=="fixed"),obj{"metadata":metadata,"body":string(b),"error":fmt.Sprint(err)})
   resp,b,err=call(host,"/","unknown",obj{},true)
   rejected:=err==nil&&resp.StatusCode==400
   check(fmt.Sprintf("%s trailers request %d",source,i),rejected==(*variant=="fixed"),obj{"rejected":rejected,"body":string(b),"error":fmt.Sprint(err)})
  }
 }
 if failures>0 {fmt.Fprintf(os.Stderr,"%d assertions failed\n",failures);os.Exit(1)}
}
