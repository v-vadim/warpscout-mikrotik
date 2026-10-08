package main

import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "fmt"
 "os"
 "os/exec"
 "os/signal"
 "path/filepath"
 "strconv"
 "strings"
 "syscall"
 "time"
)

type settings struct { args []string; confIndex int; equals bool; filename, output, state string; interval time.Duration }

// Split arguments with quotes and escapes, without executing shell syntax.
func splitArgs(s string) ([]string, error) {
 var args []string; var b strings.Builder
 var quote rune; escaped, started := false, false
 for _, c := range s {
  if escaped { b.WriteRune(c); escaped=false; started=true; continue }
  if c=='\\' && quote!='\'' { escaped=true; started=true; continue }
  if quote!=0 { if c==quote { quote=0 } else { b.WriteRune(c) }; continue }
  if c=='\'' || c=='"' { quote=c; started=true; continue }
  if c==' ' || c=='\t' || c=='\n' || c=='\r' { if started { args=append(args,b.String()); b.Reset(); started=false }; continue }
  b.WriteRune(c); started=true
 }
 if escaped || quote!=0 { return nil,fmt.Errorf("unclosed quote or escape in SCAN_COMMAND") }
 if started { args=append(args,b.String()) }
 return args,nil
}

func readSettings() (settings,error) {
 s:=settings{output:"/output",state:"/state"}
 command:=strings.TrimSpace(os.Getenv("SCAN_COMMAND"))
 if command=="" { return s,fmt.Errorf("SCAN_COMMAND is required in Envs list warpscout") }
 args,err:=splitArgs(command); if err!=nil { return s,err }
 if len(args)==0 || args[0]!="scan" { return s,fmt.Errorf("SCAN_COMMAND must start with scan, without warpscout") }
 count:=0
 for i,a:=range args {
  key,_,_:=strings.Cut(a,"=")
  switch key {
  case "-a","--a","-account","--account": return s,fmt.Errorf("account location is fixed at /state; configure Mounts")
  case "-conf","--conf":
   count++
   if strings.Contains(a,"=") { s.confIndex=i; s.equals=true; _,s.filename,_=strings.Cut(a,"=") } else {
    if i+1>=len(args) { return s,fmt.Errorf("-conf needs a filename") }; s.confIndex=i+1; s.filename=args[i+1]
   }
  }
 }
 if count!=1 { return s,fmt.Errorf("exactly one -conf filename is required") }
 if s.filename=="" || s.filename=="." || s.filename==".." || s.filename=="-" || strings.ContainsAny(s.filename,"/\\") { return s,fmt.Errorf("-conf must be a filename without a directory; mount dst=/output") }
 raw:=strings.TrimSpace(os.Getenv("INTERVAL_SECONDS"))
 if raw=="" { return s,fmt.Errorf("INTERVAL_SECONDS is required in Envs list warpscout") }
 seconds,err:=strconv.ParseInt(raw,10,64)
 if err!=nil || seconds<0 || seconds>int64((1<<63-1)/time.Second) { return s,fmt.Errorf("invalid INTERVAL_SECONDS") }
 s.args=args; s.interval=time.Duration(seconds)*time.Second
 return s,nil
}

func run(ctx context.Context,args []string,dir string) error {
 cmd:=exec.CommandContext(ctx,"/usr/local/bin/warpscout",args...)
 cmd.Dir=dir; cmd.Stdout=os.Stdout; cmd.Stderr=os.Stderr
 cmd.Cancel=func()error{return cmd.Process.Signal(syscall.SIGTERM)}
 cmd.WaitDelay=8*time.Second
 return cmd.Run()
}

type executor func(context.Context,[]string,string)error

func atomicWrite(path string,data []byte) error {
 f,err:=os.CreateTemp(filepath.Dir(path),".marker-*.tmp"); if err!=nil{return err}
 name:=f.Name(); defer os.Remove(name)
 if _,err=f.Write(data); err!=nil { f.Close(); return err }
 if err=f.Sync(); err!=nil { f.Close(); return err }
 if err=f.Close(); err!=nil{return err}
 return os.Rename(name,path)
}

func scanOnce(ctx context.Context,s settings,execute executor) error {
 if err:=os.MkdirAll(s.output,0700); err!=nil{return err}
 if err:=os.MkdirAll(s.state,0700); err!=nil{return err}
 account:=filepath.Join(s.state,"warpscout-account.json")
 if _,err:=os.Stat(account); os.IsNotExist(err) {
  fmt.Println("Registering a persistent WARP account")
  if err=execute(ctx,[]string{"register","-account",account,"-plain"},s.state); err!=nil{return fmt.Errorf("registration failed: %w",err)}
 } else if err!=nil {return err}
 f,err:=os.CreateTemp(s.output,".warpscout-*.tmp"); if err!=nil{return err}
 temp:=f.Name(); f.Close(); defer os.Remove(temp)
 args:=append([]string(nil),s.args...)
 if s.equals {args[s.confIndex]="-conf="+temp} else {args[s.confIndex]=temp}
 args=append(args,"-account",account)
 plain,report:=false,false
 for _,a:=range s.args {
  key,_,_:=strings.Cut(a,"=")
  if key=="-plain" || key=="--plain" {plain=true}
  switch key {case "-o","--o","-output","--output","-no-report","--no-report":report=true}
 }
 if !plain {args=append(args,"-plain")}; if !report {args=append(args,"-no-report")}
 fmt.Println("Scanning; target file:",filepath.Join(s.output,s.filename))
 if err=execute(ctx,args,s.state); err!=nil{return fmt.Errorf("scan failed; previous config and marker kept: %w",err)}
 if ctx.Err()!=nil{return ctx.Err()}
 info,err:=os.Stat(temp); if err!=nil{return err}; if info.Size()==0{return fmt.Errorf("empty config; previous config and marker kept")}
 // Generate the token before publication; write it only after the config.
 token:=make([]byte,16); if _,err=rand.Read(token); err!=nil{return err}
 marker:=[]byte(time.Now().UTC().Format(time.RFC3339Nano)+" "+hex.EncodeToString(token)+"\n")
 if err=os.Chmod(temp,0600); err!=nil{return err}
 if err=os.Rename(temp,filepath.Join(s.output,s.filename)); err!=nil{return err}
 if err=atomicWrite(filepath.Join(s.state,"last-success"),marker); err!=nil{return fmt.Errorf("config published, but marker could not be written: %w",err)}
 fmt.Println("Published config and success marker:",s.filename)
 return nil
}

func main() {
 syscall.Umask(0077)
 s,err:=readSettings(); if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(2)}
 ctx,cancel:=signal.NotifyContext(context.Background(),syscall.SIGTERM,syscall.SIGINT);defer cancel()
 for {
  err=scanOnce(ctx,s,run)
  if ctx.Err()!=nil{return}
  if err!=nil{fmt.Fprintln(os.Stderr,err)}
  if s.interval==0 {if err!=nil{os.Exit(1)};return}
  fmt.Printf("Next scan in %d seconds\n",int64(s.interval/time.Second))
  timer:=time.NewTimer(s.interval)
  select {case <-ctx.Done():timer.Stop();return;case <-timer.C:}
 }
}
