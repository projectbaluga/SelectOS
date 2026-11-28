package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type deviceHeartbeat struct {
	DeviceID string  `json:"device_id"`
	Hostname string  `json:"hostname"`
	Platform string  `json:"platform"`
	Version  string  `json:"version"`
	CPUPercent float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent float64 `json:"disk_percent"`
}

func migrateControlCenter(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY, hostname TEXT NOT NULL, platform TEXT NOT NULL DEFAULT '',
			version TEXT NOT NULL DEFAULT '', token_hash TEXT NOT NULL,
			remote_ip TEXT NOT NULL DEFAULT '', cpu_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
			memory_percent DOUBLE PRECISION NOT NULL DEFAULT 0, disk_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT TRUE, enrolled_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS devices_last_seen_idx ON devices(last_seen DESC)`,
	}
	for _, q := range stmts { if _, err := db.Exec(q); err != nil { return err } }
	return nil
}

func registerControlCenter(s *Server, mux *http.ServeMux) {
	if err := migrateControlCenter(s.db); err != nil { panic(fmt.Errorf("control center migration: %w", err)) }
	mux.HandleFunc("GET /dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/", http.StatusTemporaryRedirect)
	})
	mux.Handle("GET /dashboard/", dashboardHandler())
	mux.HandleFunc("POST /v1/devices/enroll", s.enrollDevice)
	mux.HandleFunc("POST /v1/devices/heartbeat", s.deviceAuth(s.heartbeat))
	mux.HandleFunc("GET /v1/admin/overview", s.auth(s.admin(s.overview)))
	mux.HandleFunc("GET /v1/admin/metrics", s.auth(s.admin(s.metrics)))
	mux.HandleFunc("GET /v1/admin/devices", s.auth(s.admin(s.listDevices)))
	mux.HandleFunc("PATCH /v1/admin/devices/{id}/status", s.auth(s.admin(s.setDeviceStatus)))
	mux.HandleFunc("GET /v1/admin/events", s.auth(s.admin(s.listEvents)))
	mux.HandleFunc("GET /v1/admin/backups", s.auth(s.admin(s.listBackups)))
}

func dashboardHandler() http.Handler {
	sub, err := fs.Sub(dashboardFiles, "dashboard")
	if err != nil { panic(err) }
	return http.StripPrefix("/dashboard/", http.FileServer(http.FS(sub)))
}

func (s *Server) enrollDevice(w http.ResponseWriter, r *http.Request) {
	secret := os.Getenv("SELECTOS_DEVICE_ENROLLMENT_TOKEN")
	provided := r.Header.Get("X-SelectOS-Enrollment-Token")
	if len(secret) < 32 || len(provided) != len(secret) || subtle.ConstantTimeCompare([]byte(secret), []byte(provided)) != 1 {
		write(w, http.StatusUnauthorized, response{"error":"device enrollment is not configured or token is invalid"}); return
	}
	var in struct { DeviceID string `json:"device_id"`; Hostname string `json:"hostname"`; Platform string `json:"platform"`; Version string `json:"version"` }
	if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in) != nil || !validDeviceID(in.DeviceID) || strings.TrimSpace(in.Hostname)=="" || len(in.Hostname)>128 {
		write(w,400,response{"error":"device_id and hostname are required"}); return
	}
	raw := make([]byte,32); if _,err:=rand.Read(raw); err!=nil { write(w,500,response{"error":"could not create device credential"}); return }
	token := hex.EncodeToString(raw); sum:=sha256.Sum256([]byte(token))
	_,err:=s.db.Exec(`INSERT INTO devices(id,hostname,platform,version,token_hash,remote_ip,last_seen)
		VALUES($1,$2,$3,$4,$5,$6,CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET hostname=EXCLUDED.hostname,platform=EXCLUDED.platform,version=EXCLUDED.version,
		token_hash=EXCLUDED.token_hash,remote_ip=EXCLUDED.remote_ip,enabled=TRUE,last_seen=CURRENT_TIMESTAMP`,
		in.DeviceID,strings.TrimSpace(in.Hostname),in.Platform,in.Version,hex.EncodeToString(sum[:]),remoteIP(r))
	if err!=nil { write(w,500,response{"error":"device registration failed"}); return }
	s.audit(in.DeviceID,"device_enrolled",r)
	write(w,201,response{"device_id":in.DeviceID,"device_token":token,"message":"Store this token securely; it is shown only once."})
}

type deviceHandler func(http.ResponseWriter,*http.Request,string)
func (s *Server) deviceAuth(next deviceHandler) http.HandlerFunc {
	return func(w http.ResponseWriter,r *http.Request) {
		p:=strings.SplitN(r.Header.Get("Authorization")," ",2)
		if len(p)!=2 || p[0]!="Bearer" || len(p[1])<32 { write(w,401,response{"error":"device bearer token required"}); return }
		sum:=sha256.Sum256([]byte(p[1])); hash:=hex.EncodeToString(sum[:])
		var id string; var enabled bool
		if err:=s.db.QueryRow("SELECT id,enabled FROM devices WHERE token_hash=$1",hash).Scan(&id,&enabled); err!=nil || !enabled {
			write(w,401,response{"error":"unknown or revoked device"}); return
		}
		next(w,r,id)
	}
}

func (s *Server) heartbeat(w http.ResponseWriter,r *http.Request,id string) {
	var in deviceHeartbeat
	if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in)!=nil || (in.DeviceID!="" && in.DeviceID!=id) || (in.CPUPercent<0 || in.CPUPercent>100 || in.MemoryPercent<0 || in.MemoryPercent>100 || in.DiskPercent<0 || in.DiskPercent>100) {
		write(w,400,response{"error":"invalid heartbeat"}); return
	}
	if strings.TrimSpace(in.Hostname)=="" || len(in.Hostname)>128 { write(w,400,response{"error":"hostname required"}); return }
	_,err:=s.db.Exec(`UPDATE devices SET hostname=$1,platform=$2,version=$3,remote_ip=$4,cpu_percent=$5,memory_percent=$6,disk_percent=$7,last_seen=CURRENT_TIMESTAMP WHERE id=$8 AND enabled=TRUE`,
		in.Hostname,in.Platform,in.Version,remoteIP(r),in.CPUPercent,in.MemoryPercent,in.DiskPercent,id)
	if err!=nil { write(w,500,response{"error":"heartbeat could not be stored"}); return }
	write(w,200,response{"accepted":true,"server_time":time.Now().UTC()})
}

func (s *Server) overview(w http.ResponseWriter,r *http.Request,c claims) {
	var devices,online int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE enabled=TRUE").Scan(&devices)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM devices WHERE enabled=TRUE AND last_seen > CURRENT_TIMESTAMP - INTERVAL '2 minutes'").Scan(&online)
	var users int; _ = s.db.QueryRow("SELECT COUNT(*) FROM users WHERE enabled=TRUE").Scan(&users)
	m:=hostMetrics()
	write(w,200,response{"server":m,"devices":response{"registered":devices,"online":online,"offline":devices-online},"users_enabled":users,"database":"connected","as_of":time.Now().UTC()})
}
func (s *Server) metrics(w http.ResponseWriter,r *http.Request,c claims) { write(w,200,hostMetrics()) }
func hostMetrics() response {
	out:=response{"hostname":hostname(),"os":runtime.GOOS,"architecture":runtime.GOARCH,"go_version":runtime.Version(),"cpus":runtime.NumCPU(),"as_of":time.Now().UTC()}
	if b,err:=os.ReadFile("/proc/meminfo"); err==nil {
		var total,available uint64
		for _,line:=range strings.Split(string(b),"\n") { f:=strings.Fields(line); if len(f)>=2 { n,_:=strconv.ParseUint(f[1],10,64); switch strings.TrimSuffix(f[0],":") { case "MemTotal": total=n; case "MemAvailable": available=n } } }
		if total>0 { out["memory_total_mb"]=total/1024; out["memory_used_percent"]=roundPct(float64(total-available)/float64(total)*100) }
	}
	if b,err:=os.ReadFile("/proc/uptime"); err==nil { f:=strings.Fields(string(b)); if len(f)>0 { seconds,_:=strconv.ParseFloat(f[0],64); out["uptime_seconds"]=int64(seconds) } }
	var st syscall.Statfs_t
	if err:=syscall.Statfs("/",&st); err==nil && st.Blocks>0 { used:=st.Blocks-st.Bfree; out["disk_total_gb"]=uint64(st.Blocks)*uint64(st.Bsize)/1024/1024/1024; out["disk_used_percent"]=roundPct(float64(used)/float64(st.Blocks)*100) }
	if b,err:=os.ReadFile("/proc/loadavg"); err==nil { out["load_average"]=strings.Fields(string(b))[:3] }
	return out
}
func hostname() string { h,_:=os.Hostname(); return h }
func roundPct(v float64) float64 { return float64(int(v*10+0.5))/10 }
func remoteIP(r *http.Request) string { host,_,err:=net.SplitHostPort(r.RemoteAddr); if err!=nil { return r.RemoteAddr }; return host }
func validDeviceID(v string) bool { if len(v)<3||len(v)>64{return false}; for _,r:=range v { if !(r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='-'||r=='_'||r=='.'){return false} }; return true }

func (s *Server) listDevices(w http.ResponseWriter,r *http.Request,c claims) {
	rows,err:=s.db.Query(`SELECT id,hostname,platform,version,remote_ip,cpu_percent,memory_percent,disk_percent,enabled,enrolled_at,last_seen,
		(last_seen > CURRENT_TIMESTAMP - INTERVAL '2 minutes') AS online FROM devices ORDER BY hostname`)
	if err!=nil { write(w,500,response{"error":"database error"});return }; defer rows.Close()
	out:=[]response{}
	for rows.Next(){var id,host,platform,version,ip string;var cpu,mem,disk float64;var enabled,online bool;var enrolled time.Time;var seen sql.NullTime
		if rows.Scan(&id,&host,&platform,&version,&ip,&cpu,&mem,&disk,&enabled,&enrolled,&seen,&online)==nil {
			item:=response{"id":id,"hostname":host,"platform":platform,"version":version,"ip":ip,"cpu_percent":cpu,"memory_percent":mem,"disk_percent":disk,"enabled":enabled,"online":online,"enrolled_at":enrolled}
			if seen.Valid { item["last_seen"]=seen.Time }; out=append(out,item)
		}
	}
	write(w,200,response{"devices":out})
}
func (s *Server) setDeviceStatus(w http.ResponseWriter,r *http.Request,c claims) {
	id:=r.PathValue("id"); if !validDeviceID(id){write(w,400,response{"error":"invalid device id"});return}
	var in struct{Enabled bool `json:"enabled"`}; if json.NewDecoder(http.MaxBytesReader(w,r.Body,1024)).Decode(&in)!=nil {write(w,400,response{"error":"invalid request"});return}
	res,err:=s.db.Exec("UPDATE devices SET enabled=$1,token_hash=CASE WHEN $1 THEN token_hash ELSE '' END WHERE id=$2",in.Enabled,id)
	if err!=nil {write(w,500,response{"error":"database error"});return}; n,_:=res.RowsAffected(); if n==0{write(w,404,response{"error":"device not found"});return}
	s.audit(id,"device_status_changed",r);write(w,200,response{"updated":true})
}
func (s *Server) listEvents(w http.ResponseWriter,r *http.Request,c claims) {
	rows,err:=s.db.Query("SELECT username,event,remote_ip,created_at FROM audit ORDER BY created_at DESC LIMIT 100")
	if err!=nil {write(w,500,response{"error":"database error"});return};defer rows.Close()
	out:=[]response{};for rows.Next(){var u,e,ip string;var at time.Time;if rows.Scan(&u,&e,&ip,&at)==nil{out=append(out,response{"actor":u,"event":e,"ip":ip,"at":at})}}
	write(w,200,response{"events":out})
}
func (s *Server) listBackups(w http.ResponseWriter,r *http.Request,c claims) {
	dir:=os.Getenv("SELECTOS_BACKUP_DIR");if dir==""{dir="/var/lib/selectos/backups"}
	entries,err:=os.ReadDir(dir);if errors.Is(err,os.ErrNotExist){write(w,200,response{"backups":[]response{},"directory":dir,"configured":false});return};if err!=nil{write(w,500,response{"error":"backup directory unavailable"});return}
	out:=[]response{};for _,e:=range entries{if e.IsDir()||!(strings.HasSuffix(e.Name(),".sql")||strings.HasSuffix(e.Name(),".sql.gz")){continue};info,err:=e.Info();if err==nil{out=append(out,response{"name":filepath.Base(e.Name()),"size_bytes":info.Size(),"modified_at":info.ModTime()})}}
	write(w,200,response{"backups":out,"directory":dir,"configured":true})
}
