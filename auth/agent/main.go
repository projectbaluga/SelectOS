package main
import("bytes";"context";"encoding/json";"errors";"fmt";"net/http";"os";"os/signal";"path/filepath";"runtime";"strconv";"strings";"syscall";"time")
type state struct{DeviceToken string}
type config struct{server,secret,id,path string}
var previousCPUTotal, previousCPUIdle uint64
func main(){
 c:=config{strings.TrimRight(os.Getenv("SELECTOS_SERVER_URL"),"/"),os.Getenv("SELECTOS_ENROLLMENT_TOKEN"),os.Getenv("SELECTOS_DEVICE_ID"),env("SELECTOS_STATE_FILE","/var/lib/selectos-agent/credentials.json")}
 if c.server==""||c.id==""{fatal(errors.New("SELECTOS_SERVER_URL and SELECTOS_DEVICE_ID are required"))}
 if !strings.HasPrefix(c.server,"http://")&&!strings.HasPrefix(c.server,"https://"){fatal(errors.New("server URL must use http or https"))}
 client:=&http.Client{Timeout:8*time.Second};s,e:=load(c.path);if e!=nil{fatal(e)}
 if s.DeviceToken==""{if c.secret==""{fatal(errors.New("SELECTOS_ENROLLMENT_TOKEN required for first enrollment"))};s,e=enroll(client,c);if e!=nil{fatal(e)};if e=save(c.path,s);e!=nil{fatal(e)}}
 host,_:=os.Hostname();tick:=time.NewTicker(30*time.Second);defer tick.Stop();ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 for{if e:=heartbeat(client,c,s,host);e!=nil{logf("heartbeat failed: %v",e)};select{case<-ctx.Done():return;case<-tick.C:}}
}
func enroll(cl *http.Client,c config)(state,error){h,_:=os.Hostname();b,_:=json.Marshal(map[string]string{"device_id":c.id,"hostname":h,"platform":runtime.GOOS+"/"+runtime.GOARCH,"version":"agent-0.1"});r,e:=http.NewRequest("POST",c.server+"/v1/devices/enroll",bytes.NewReader(b));if e!=nil{return state{},e};r.Header.Set("Content-Type","application/json");r.Header.Set("X-SelectOS-Enrollment-Token",c.secret);res,e:=cl.Do(r);if e!=nil{return state{},e};defer res.Body.Close();var out map[string]any;if e=json.NewDecoder(res.Body).Decode(&out);e!=nil{return state{},e};if res.StatusCode!=201{return state{},fmt.Errorf("enrollment rejected: HTTP %d",res.StatusCode)};t,_:=out["device_token"].(string);if len(t)<32{return state{},errors.New("invalid device token returned")};return state{t},nil}
func heartbeat(cl *http.Client,c config,s state,h string)error{b,_:=json.Marshal(map[string]any{"device_id":c.id,"hostname":h,"platform":runtime.GOOS+"/"+runtime.GOARCH,"version":"agent-0.1","cpu_percent":cpu(),"memory_percent":mem(),"disk_percent":disk()});r,e:=http.NewRequest("POST",c.server+"/v1/devices/heartbeat",bytes.NewReader(b));if e!=nil{return e};r.Header.Set("Content-Type","application/json");r.Header.Set("Authorization","Bearer "+s.DeviceToken);res,e:=cl.Do(r);if e!=nil{return e};defer res.Body.Close();if res.StatusCode!=200{return fmt.Errorf("heartbeat rejected: HTTP %d",res.StatusCode)};logf("heartbeat accepted");return nil}
func cpu()float64{b,e:=os.ReadFile("/proc/stat");if e!=nil{return 0};fields:=strings.Fields(strings.SplitN(string(b),"\\n",2)[0]);if len(fields)<5{return 0};var total,idle uint64;for i,v:=range fields[1:]{n,err:=strconv.ParseUint(v,10,64);if err!=nil{continue};total+=n;if i==3||i==4{idle+=n}};if previousCPUTotal==0{previousCPUTotal=total;previousCPUIdle=idle;return 0};dt:=total-previousCPUTotal;di:=idle-previousCPUIdle;previousCPUTotal=total;previousCPUIdle=idle;if dt==0{return 0};return pct(float64(dt-di)/float64(dt)*100)}
func mem()float64{b,e:=os.ReadFile("/proc/meminfo");if e!=nil{return 0};var total,avail float64;for _,l:=range strings.Split(string(b),"\n"){f:=strings.Fields(l);if len(f)>1{var n float64;fmt.Sscan(f[1],&n);switch strings.TrimSuffix(f[0],":"){case"MemTotal":total=n;case"MemAvailable":avail=n}}};if total==0{return 0};return pct((total-avail)/total*100)}
func disk()float64{var s syscall.Statfs_t;if syscall.Statfs("/",&s)!=nil||s.Blocks==0{return 0};return pct(float64(s.Blocks-s.Bfree)/float64(s.Blocks)*100)}
func pct(v float64)float64{return float64(int(v*10+0.5))/10}
func load(p string)(state,error){b,e:=os.ReadFile(p);if os.IsNotExist(e){return state{},nil};if e!=nil{return state{},e};var x map[string]string;e=json.Unmarshal(b,&x);return state{x["device_token"]},e}
func save(p string,s state)error{if e:=os.MkdirAll(filepath.Dir(p),0700);e!=nil{return e};b,e:=json.Marshal(map[string]string{"device_token":s.DeviceToken});if e!=nil{return e};t:=p+".tmp";if e=os.WriteFile(t,b,0600);e!=nil{return e};if e=os.Rename(t,p);e!=nil{return e};return os.Chmod(p,0600)}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
func fatal(e error){fmt.Fprintln(os.Stderr,e);os.Exit(1)}
func logf(f string,a ...any){fmt.Fprintf(os.Stderr,time.Now().UTC().Format(time.RFC3339)+" "+f+"\n",a...)}
