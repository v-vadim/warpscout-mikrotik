package main

import("context";"fmt";"os";"path/filepath";"strings";"testing")

func TestRequiredSettings(t *testing.T){
 t.Setenv("SCAN_COMMAND","");t.Setenv("INTERVAL_SECONDS","21600")
 if _,e:=readSettings();e==nil{t.Fatal("missing command accepted")}
 t.Setenv("SCAN_COMMAND","scan -p awg -tg-only -exclude-node DME -conf warp_ru_5.conf")
 t.Setenv("INTERVAL_SECONDS","");if _,e:=readSettings();e==nil{t.Fatal("missing interval accepted")}
 t.Setenv("INTERVAL_SECONDS","21600");s,e:=readSettings();if e!=nil||s.filename!="warp_ru_5.conf"{t.Fatal(s,e)}
 t.Setenv("SCAN_COMMAND",`scan -conf="warp name.conf"`);s,e=readSettings();if e!=nil||!s.equals||s.filename!="warp name.conf"{t.Fatal(s,e)}
 for _,c:=range []string{"scan -conf ../bad","scan -conf a -conf b","scan -conf a -account secret","scan -conf -"}{t.Setenv("SCAN_COMMAND",c);if _,e:=readSettings();e==nil{t.Fatal("accepted",c)}}
}

func TestPublicationAndAccount(t *testing.T){
 for _,mode:=range []string{"success","failure","empty"}{t.Run(mode,func(t *testing.T){
  root:=t.TempDir();state:=filepath.Join(root,"state");os.Mkdir(state,0700)
  account:=filepath.Join(state,"warpscout-account.json");os.WriteFile(account,[]byte("existing"),0600)
  target:=filepath.Join(root,"a.conf");os.WriteFile(target,[]byte("old"),0600)
  marker:=filepath.Join(state,"last-success");os.WriteFile(marker,[]byte("old-token"),0600)
  s:=settings{args:[]string{"scan","-conf","a.conf"},confIndex:2,filename:"a.conf",output:root,state:state}
  registered:=0
  fake:=func(ctx context.Context,args []string,dir string)error{
   if args[0]=="register"{registered++;return os.WriteFile(account,[]byte("new-account"),0600)}
   if mode=="failure"{return fmt.Errorf("failed")};if mode=="empty"{return nil}
   return os.WriteFile(args[2],[]byte("new-config"),0600)
  }
  e:=scanOnce(context.Background(),s,fake);data,_:=os.ReadFile(target);token,_:=os.ReadFile(marker)
  if registered!=0{t.Fatal("existing account replaced")}
  if mode=="success"{
   if e!=nil||string(data)!="new-config"||string(token)=="old-token"{t.Fatal(e,string(data),string(token))}
   oldToken:=string(token);os.Remove(account)
   if e=scanOnce(context.Background(),s,fake);e!=nil{t.Fatal(e)}
   token,_=os.ReadFile(marker);if registered!=1||string(token)==oldToken{t.Fatal("account/token not regenerated")}
  }else if e==nil||string(data)!="old"||string(token)!="old-token"{t.Fatal("failure changed publication",e)}
  entries,_:=os.ReadDir(root);for _,entry:=range entries{if strings.HasSuffix(entry.Name(),".tmp"){t.Fatal("temp leaked")}}
 })}
}

func TestRegistrationFailure(t *testing.T){
 root:=t.TempDir();s:=settings{output:root,state:root,filename:"a.conf",args:[]string{"scan","-conf","a.conf"},confIndex:2}
 os.WriteFile(filepath.Join(root,"last-success"),[]byte("old"),0600)
 e:=scanOnce(context.Background(),s,func(context.Context,[]string,string)error{return fmt.Errorf("no network")})
 data,_:=os.ReadFile(filepath.Join(root,"last-success"));if e==nil||string(data)!="old"{t.Fatal(e)}
}
