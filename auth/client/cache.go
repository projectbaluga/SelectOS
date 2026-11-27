package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/argon2"
)

const cacheFile = "offline-cache.enc"
const keyFile = "device.key"

type offlineRecord struct {
	Username string `json:"username"`
	Role string `json:"role"`
	Server string `json:"server"`
	Salt []byte `json:"salt"`
	Verifier []byte `json:"verifier"`
	CreatedAt int64 `json:"created_at"`
	ExpiresAt int64 `json:"expires_at"`
}

// SaveOfflineCache must only be called after the server has accepted an online login.
// The device key is a local, root-owned secret; protect the containing directory.
func SaveOfflineCache(dir, username, password, server, role string, lifetime time.Duration) error {
	if len(password)<1 || username=="" { return errors.New("missing credentials") }
	if lifetime<=0 || lifetime>72*time.Hour { return errors.New("offline cache lifetime must be between 0 and 72 hours") }
	if err:=os.MkdirAll(dir,0700);err!=nil{return err}
	keyPath:=filepath.Join(dir,keyFile)
	key,err:=os.ReadFile(keyPath)
	if os.IsNotExist(err){key=make([]byte,32);if _,err=rand.Read(key);err!=nil{return err};if err=os.WriteFile(keyPath,key,0600);err!=nil{return err}} else if err!=nil{return err}
	if len(key)!=32{return errors.New("invalid device key; remove cache and re-enroll this device")}
	salt:=make([]byte,16);if _,err=rand.Read(salt);err!=nil{return err}
	verifier:=argon2.IDKey([]byte(password),salt,3,64*1024,2,32)
	now:=time.Now()
	record:=offlineRecord{Username:username,Role:role,Server:server,Salt:salt,Verifier:verifier,CreatedAt:now.Unix(),ExpiresAt:now.Add(lifetime).Unix()}
	plain,err:=json.Marshal(record);if err!=nil{return err}
	block,err:=aes.NewCipher(key);if err!=nil{return err};gcm,err:=cipher.NewGCM(block);if err!=nil{return err}
	nonce:=make([]byte,gcm.NonceSize());if _,err=rand.Read(nonce);err!=nil{return err}
	sealed:=gcm.Seal(nonce,nonce,plain,[]byte("SelectOS-offline-cache-v1"))
	tmp:=filepath.Join(dir,cacheFile+".tmp")
	if err=os.WriteFile(tmp,sealed,0600);err!=nil{return err}
	return os.Rename(tmp,filepath.Join(dir,cacheFile))
}

func VerifyOfflineCache(dir, username, password string)(string,error){
	key,err:=os.ReadFile(filepath.Join(dir,keyFile));if err!=nil{return "",errors.New("this device has no enrolled offline cache")}
	if len(key)!=32{return "",errors.New("invalid device key")}
	sealed,err:=os.ReadFile(filepath.Join(dir,cacheFile));if err!=nil{return "",errors.New("offline cache unavailable")}
	block,err:=aes.NewCipher(key);if err!=nil{return "",err};gcm,err:=cipher.NewGCM(block);if err!=nil{return "",err}
	if len(sealed)<gcm.NonceSize(){return "",errors.New("corrupt offline cache")}
	nonce:=sealed[:gcm.NonceSize()];plain,err:=gcm.Open(nil,nonce,sealed[gcm.NonceSize():],[]byte("SelectOS-offline-cache-v1"));if err!=nil{return "",errors.New("offline cache authentication failed")}
	var record offlineRecord;if err=json.Unmarshal(plain,&record);err!=nil{return "",err}
	if record.Username!=username{return "",errors.New("username does not match this device cache")}
	if time.Now().Unix()>record.ExpiresAt{return "",errors.New("offline cache expired; reconnect to the LAN server")}
	if len(record.Salt)!=16||len(record.Verifier)!=32{return "",fmt.Errorf("invalid offline verifier")}
	candidate:=argon2.IDKey([]byte(password),record.Salt,3,64*1024,2,32)
	if subtle.ConstantTimeCompare(candidate,record.Verifier)!=1{return "",errors.New("incorrect password")}
	return record.Role,nil
}
