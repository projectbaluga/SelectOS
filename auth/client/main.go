package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type loginRequest struct { Username string `json:"username"`; Password string `json:"password"` }
type loginResponse struct { AccessToken string `json:"access_token"`; TokenType string `json:"token_type"`; ExpiresAt int64 `json:"expires_at"`; User struct { ID int64 `json:"id"`; Username string `json:"username"`; Role string `json:"role"` } `json:"user"` }

func main() {
	if len(os.Args)<2 { usage(); os.Exit(2) }
	switch os.Args[1] {
	case "login": onlineLogin(os.Args[2:])
	case "offline-check": offlineCheck(os.Args[2:])
	default: usage(); os.Exit(2)
	}
}
func usage(){fmt.Fprintln(os.Stderr,"SelectOS auth client\n  login --server http://debian-host:8088 --username USER\n  offline-check --username USER\nPassword is read from SELECTOS_PASSWORD; cache key is stored locally with mode 0600.")}
func onlineLogin(args []string) {
	fs:=flag.NewFlagSet("login",flag.ExitOnError); server:=fs.String("server", "http://127.0.0.1:8088","Authentication API base URL"); username:=fs.String("username","","Account username"); cacheDir:=fs.String("cache-dir",defaultCacheDir(),"Cache directory"); _=fs.Parse(args)
	password:=os.Getenv("SELECTOS_PASSWORD")
	if *username==""||password==""{fmt.Fprintln(os.Stderr,"Set --username and SELECTOS_PASSWORD; do not put passwords in shell history.");os.Exit(2)}
	if !strings.HasPrefix(*server,"https://")&&!strings.HasPrefix(*server,"http://"){fmt.Fprintln(os.Stderr,"server must be an http(s) URL");os.Exit(2)}
	body,_:=json.Marshal(loginRequest{Username:*username,Password:password})
	req,err:=http.NewRequest(http.MethodPost,strings.TrimRight(*server,"/")+"/v1/auth/login",bytes.NewReader(body));if err!=nil{fatal(err)}
	req.Header.Set("Content-Type","application/json")
	client:=&http.Client{Timeout:10*time.Second}
	resp,err:=client.Do(req);if err!=nil{fatal(fmt.Errorf("cannot reach auth server: %w",err))};defer resp.Body.Close()
	data,_:=io.ReadAll(io.LimitReader(resp.Body,1<<20))
	if resp.StatusCode!=http.StatusOK{fmt.Fprintf(os.Stderr,"Server rejected login (HTTP %d): %s\n",resp.StatusCode,strings.TrimSpace(string(data)));os.Exit(1)}
	var result loginResponse;if err=json.Unmarshal(data,&result);err!=nil||result.AccessToken==""{fatal(fmt.Errorf("invalid response from auth server"))}
	if err=SaveOfflineCache(*cacheDir,*username,password,*server,result.User.Role,72*time.Hour);err!=nil{fatal(fmt.Errorf("online login succeeded, but offline cache could not be saved: %w",err))}
	fmt.Printf("Online login successful: %s (%s). Offline cache valid for up to 72 hours on this device.\n",result.User.Username,result.User.Role)
}
func offlineCheck(args []string){
	fs:=flag.NewFlagSet("offline-check",flag.ExitOnError);username:=fs.String("username","","Account username");cacheDir:=fs.String("cache-dir",defaultCacheDir(),"Cache directory");_=fs.Parse(args)
	password:=os.Getenv("SELECTOS_PASSWORD");if *username==""||password==""{fmt.Fprintln(os.Stderr,"Set --username and SELECTOS_PASSWORD.");os.Exit(2)}
	role,err:=VerifyOfflineCache(*cacheDir,*username,password);if err!=nil{fmt.Fprintln(os.Stderr,"Offline login denied:",err);os.Exit(1)}
	fmt.Printf("Offline login accepted: %s (%s). Server policy changes may not be known until reconnect.\n",*username,role)
}
func defaultCacheDir()string{if v:=os.Getenv("SELECTOS_AUTH_CACHE");v!=""{return v};return "/var/lib/selectos-auth-client"}
func fatal(err error){fmt.Fprintln(os.Stderr,err);os.Exit(1)}
func init(){_ = filepath.Separator}
