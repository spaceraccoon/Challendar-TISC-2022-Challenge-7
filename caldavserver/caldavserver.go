package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/webdav"
)

// Copied from https://github.com/foomo/htpasswd
// HashedPasswords name => hash
type HashedPasswords map[string]string

// HashAlgorithm enum for hashing algorithms
type HashAlgorithm string

const (
	// PasswordSeparator separates passwords from hashes
	PasswordSeparator = ":"
	// LineSeparator separates password records
	LineSeparator = "\n"
)

// MaxHtpasswdFilesize if your htpassd file is larger than 8MB, then your are doing it wrong
const MaxHtpasswdFilesize = 8 * 1024 * 1024

// ErrNotExist is the error returned when a user does not exist.
var ErrNotExist = errors.New("user did not exist in file")

// Bytes bytes representation
func (hp HashedPasswords) Bytes() (passwordBytes []byte) {
	passwordBytes = []byte{}
	for name, hash := range hp {
		passwordBytes = append(passwordBytes, []byte(name+PasswordSeparator+hash+LineSeparator)...)
	}
	return passwordBytes
}

// WriteToFile put them to a file will be overwritten or created
func (hp HashedPasswords) WriteToFile(file string) error {
	return ioutil.WriteFile(file, hp.Bytes(), 0644)
}

// ParseHtpasswdFile load a htpasswd file
func ParseHtpasswdFile(file string) (passwords HashedPasswords, err error) {
	htpasswdBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	if len(htpasswdBytes) > MaxHtpasswdFilesize {
		err = errors.New("this file is too large, use a database instead")
		return
	}
	return ParseHtpasswd(htpasswdBytes)
}

// ParseHtpasswd parse htpasswd bytes
func ParseHtpasswd(htpasswdBytes []byte) (passwords HashedPasswords, err error) {
	lines := strings.Split(string(htpasswdBytes), LineSeparator)
	passwords = make(map[string]string)
	for lineNumber, line := range lines {
		// scan lines
		line = strings.Trim(line, " ")
		if len(line) == 0 {
			// skipping empty lines
			continue
		}
		parts := strings.Split(line, PasswordSeparator)
		if len(parts) != 2 {
			err = errors.New(fmt.Sprintln("invalid line", lineNumber+1, "unexpected number of parts split by", PasswordSeparator, len(parts), "instead of 2 in\"", line, "\""))
			return
		}
		for i, part := range parts {
			parts[i] = strings.Trim(part, " ")
		}
		_, alreadyExists := passwords[parts[0]]
		if alreadyExists {
			err = errors.New("invalid htpasswords file - user " + parts[0] + " was already defined")
			return
		}
		passwords[parts[0]] = parts[1]
	}
	return
}

func checkIsAuthorized(req *http.Request) error {
	// should already be authorized
	username, _, _ := req.BasicAuth()
	urlParts := strings.Split(req.URL.Path, "/")
	// users can only access their own resources
	if username != urlParts[1] || len(urlParts) > 4 {
		return ErrNotExist
	}
	return nil
}

func main() {
	// Backward-compatible with with our current Radicale files
	passwords, _ := ParseHtpasswdFile("/etc/radicale/users")
	fs := &webdav.Handler{
		FileSystem: webdav.Dir("/var/lib/radicale/collections/collection-root"),
		LockSystem: webdav.NewMemLS(),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		username, password, ok := req.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="CalDavServer - Password Required"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err := bcrypt.CompareHashAndPassword([]byte(passwords[username]), []byte(password))
		if err != nil {
			http.Error(w, "Access to the requested resource forbidden.", http.StatusUnauthorized)
			return
		}

		err = checkIsAuthorized(req)
		if err != nil {
			http.Error(w, "Access to the requested resource forbidden.", http.StatusUnauthorized)
			return
		}

		switch req.Method {
		// To update to CalDAV RFC... been taking too many coffee breaks
		case "PROPFIND", "PROPPATCH", "MKCALENDAR", "MKCOL", "REPORT":
			http.Error(w, "Method not implemented.", http.StatusNotImplemented)
			return
		}

		fs.ServeHTTP(w, req)
	})
	http.ListenAndServe(":4000", nil)

}
