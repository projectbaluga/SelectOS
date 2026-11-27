package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

type Server struct { db *sql.DB; tokenKey []byte; issuer string }
type claims struct { UserID int64 `json:"uid"`; Username string `json:"sub"`; Role string `json:"role"`; Expires int64 `json:"exp"` }
type credentials struct { Username string `json:"username"`; Password string `json:"password"` }
type userInput struct { Username string `json:"username"`; Password string `json:"password"`; Role string `json:"role"` }
type response map[string]any

func main() {
	addr := env("SELECTOS_AUTH_ADDR", "0.0.0.0:8088")
	databaseURL := os.Getenv("SELECTOS_DATABASE_URL"); if databaseURL == "" { log.Fatal("SELECTOS_DATABASE_URL is required") }
	keyText := os.Getenv("SELECTOS_AUTH_TOKEN_KEY")
	if len(keyText) < 32 { log.Fatal("SELECTOS_AUTH_TOKEN_KEY must be a random secret of at least 32 characters") }
	db, err := sql.Open("pgx", databaseURL)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err = migrate(db); err != nil { log.Fatal(err) }
	s := &Server{db:db, tokenKey:[]byte(keyText), issuer:"selectos-local"}
	if err = s.bootstrapAdmin(); err != nil { log.Fatal(err) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("GET /v1/auth/me", s.auth(s.me))
	mux.HandleFunc("GET /v1/admin/users", s.auth(s.admin(s.listUsers)))
	mux.HandleFunc("POST /v1/admin/users", s.auth(s.admin(s.createUser)))
	mux.HandleFunc("PATCH /v1/admin/users/{id}/status", s.auth(s.admin(s.setUserStatus)))
	mux.HandleFunc("POST /v1/admin/users/{id}/password", s.auth(s.admin(s.resetPassword)))

	srv := &http.Server{Addr:addr, Handler:securityHeaders(mux), ReadHeaderTimeout:5*time.Second, ReadTimeout:10*time.Second, WriteTimeout:15*time.Second, IdleTimeout:60*time.Second}
	log.Printf("SelectOS auth server listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY, username TEXT NOT NULL,
		password_hash TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'user',
		enabled BOOLEAN NOT NULL DEFAULT TRUE, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS audit (
		id BIGSERIAL PRIMARY KEY, username TEXT NOT NULL, event TEXT NOT NULL,
		remote_ip TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	 );
	CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_idx ON users (LOWER(username));`)
	return err
}
func (s *Server) bootstrapAdmin() error {
	u, p := os.Getenv("SELECTOS_ADMIN_USER"), os.Getenv("SELECTOS_ADMIN_PASSWORD")
	if u == "" && p == "" { return nil }
	if u == "" || len(p) < 12 { return errors.New("set both SELECTOS_ADMIN_USER and SELECTOS_ADMIN_PASSWORD (minimum 12 characters)") }
	hash, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost); if err != nil { return err }
	_, err = s.db.Exec("INSERT INTO users(username,password_hash,role) VALUES($1,$2,'admin') ON CONFLICT DO NOTHING", u, string(hash))
	return err
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) { write(w,200,response{"status":"ok","service":"selectos-auth","time":time.Now().UTC()}) }
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in) != nil || len(in.Username)>64 || len(in.Password)>256 { write(w,400,response{"error":"invalid request"}); return }
	var c claims; var hash, role string; var enabled bool
	err := s.db.QueryRow("SELECT id,username,password_hash,role,enabled FROM users WHERE LOWER(username)=LOWER($1)",strings.TrimSpace(in.Username)).Scan(&c.UserID,&c.Username,&hash,&role,&enabled)
	ok := err == nil && enabled && bcrypt.CompareHashAndPassword([]byte(hash),[]byte(in.Password)) == nil
	if !ok { s.audit(in.Username,"login_failed",r); write(w,401,response{"error":"invalid credentials"}); return }
	c.Role=role; c.Expires=time.Now().Add(8*time.Hour).Unix()
	token, err := s.sign(c); if err != nil { write(w,500,response{"error":"token error"}); return }
	s.audit(c.Username,"login_success",r)
	write(w,200,response{"access_token":token,"token_type":"Bearer","expires_at":c.Expires,"user":response{"id":c.UserID,"username":c.Username,"role":c.Role}})
}
func (s *Server) me(w http.ResponseWriter,r *http.Request,c claims) { write(w,200,response{"id":c.UserID,"username":c.Username,"role":c.Role,"expires_at":c.Expires}) }
func (s *Server) listUsers(w http.ResponseWriter,r *http.Request,c claims) {
	rows,err:=s.db.Query("SELECT id,username,role,enabled,created_at FROM users ORDER BY username"); if err!=nil { write(w,500,response{"error":"database error"}); return }; defer rows.Close()
	out:=[]response{}; for rows.Next(){var id int64;var u,role,created string;var enabled bool;if rows.Scan(&id,&u,&role,&enabled,&created)==nil{out=append(out,response{"id":id,"username":u,"role":role,"enabled":enabled,"created_at":created})}}
	write(w,200,response{"users":out})
}
func (s *Server) createUser(w http.ResponseWriter,r *http.Request,c claims) {
	var in userInput; if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in)!=nil || !validUser(in.Username) || len(in.Password)<12 { write(w,400,response{"error":"username must be 3-32 safe characters; password must be at least 12 characters"});return }
	if in.Role!="admin" {in.Role="user"}
	h,err:=bcrypt.GenerateFromPassword([]byte(in.Password),bcrypt.DefaultCost);if err!=nil{write(w,500,response{"error":"hash failure"});return}
	res,err:=s.db.Exec("INSERT INTO users(username,password_hash,role) VALUES($1,$2,$3)",in.Username,string(h),in.Role);if err!=nil{write(w,409,response{"error":"username already exists or database error"});return}
	id,_:=res.LastInsertId();s.audit(in.Username,"user_created",r);write(w,201,response{"id":id,"username":in.Username,"role":in.Role})
}
func (s *Server) setUserStatus(w http.ResponseWriter,r *http.Request,c claims) {
	id,err:=strconv.ParseInt(r.PathValue("id"),10,64);if err!=nil{write(w,400,response{"error":"invalid user id"});return}
	var in struct{Enabled bool `json:"enabled"`};if json.NewDecoder(r.Body).Decode(&in)!=nil{write(w,400,response{"error":"invalid request"});return}
	if id==c.UserID && !in.Enabled{write(w,400,response{"error":"cannot disable your own admin account"});return}
	res,err:=s.db.Exec("UPDATE users SET enabled=$1 WHERE id=$2",in.Enabled,id);if err!=nil{write(w,500,response{"error":"database error"});return};n,_:=res.RowsAffected();if n==0{write(w,404,response{"error":"user not found"});return}
	write(w,200,response{"updated":true})
}
func (s *Server) resetPassword(w http.ResponseWriter,r *http.Request,c claims) {
	id,err:=strconv.ParseInt(r.PathValue("id"),10,64);if err!=nil{write(w,400,response{"error":"invalid user id"});return}
	var in struct{Password string `json:"password"`};if json.NewDecoder(http.MaxBytesReader(w,r.Body,4096)).Decode(&in)!=nil||len(in.Password)<12{write(w,400,response{"error":"password must be at least 12 characters"});return}
	h,err:=bcrypt.GenerateFromPassword([]byte(in.Password),bcrypt.DefaultCost);if err!=nil{write(w,500,response{"error":"hash failure"});return}
	res,err:=s.db.Exec("UPDATE users SET password_hash=$1 WHERE id=$2",string(h),id);if err!=nil{write(w,500,response{"error":"database error"});return};n,_:=res.RowsAffected();if n==0{write(w,404,response{"error":"user not found"});return};write(w,200,response{"updated":true})
}
type handler func(http.ResponseWriter,*http.Request,claims)
func (s *Server) auth(next handler) http.HandlerFunc { return func(w http.ResponseWriter,r *http.Request){p:=strings.SplitN(r.Header.Get("Authorization")," ",2);if len(p)!=2||p[0]!="Bearer"{write(w,401,response{"error":"missing bearer token"});return};c,err:=s.verify(p[1]);if err!=nil{write(w,401,response{"error":"invalid or expired token"});return};next(w,r,c)} }
func (s *Server) admin(next handler) handler { return func(w http.ResponseWriter,r *http.Request,c claims){if subtle.ConstantTimeCompare([]byte(c.Role),[]byte("admin"))!=1{write(w,403,response{"error":"admin required"});return};next(w,r,c)} }
func (s *Server) sign(c claims)(string,error){b,e:=json.Marshal(c);if e!=nil{return "",e};mac:=hmac256(s.tokenKey,b);return base64.RawURLEncoding.EncodeToString(b)+"."+base64.RawURLEncoding.EncodeToString(mac),nil}
func (s *Server) verify(t string)(claims,error){var c claims;p:=strings.Split(t,".");if len(p)!=2{return c,errors.New("bad token")};b,e:=base64.RawURLEncoding.DecodeString(p[0]);if e!=nil{return c,e};sig,e:=base64.RawURLEncoding.DecodeString(p[1]);if e!=nil{return c,e};if subtle.ConstantTimeCompare(sig,hmac256(s.tokenKey,b))!=1{return c,errors.New("bad signature")};if json.Unmarshal(b,&c)!=nil||c.Expires<time.Now().Unix(){return c,errors.New("expired")};return c,nil}
func hmac256(key,data []byte)[]byte{h:=hmac.New(sha256.New,key);h.Write(data);return h.Sum(nil)}
func (s *Server) audit(u,event string,r *http.Request){ip,_,_:=net.SplitHostPort(r.RemoteAddr);_,_=s.db.Exec("INSERT INTO audit(username,event,remote_ip) VALUES($1,$2,$3)",u,event,ip)}
func validUser(u string)bool{if len(u)<3||len(u)>32{return false};for _,r:=range u{if !(r>='a'&&r<='z'||r>='A'&&r<='Z'||r>='0'&&r<='9'||r=='_'||r=='-'||r=='.'){return false}};return true}
func write(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_=json.NewEncoder(w).Encode(v)}
func securityHeaders(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Header().Set("X-Content-Type-Options","nosniff");w.Header().Set("Cache-Control","no-store");next.ServeHTTP(w,r)})}
func env(k,d string)string{if v:=os.Getenv(k);v!=""{return v};return d}
