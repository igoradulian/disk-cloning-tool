package forensic

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type HashResults struct {
	MD5    string
	SHA1   string
	SHA256 string
}

// ComputeFileHashes computes hashes for the given file path
func ComputeFileHashes(filePath string) (*HashResults, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hMd5 := md5.New()
	hSha1 := sha1.New()
	hSha256 := sha256.New()
	multiWriter := io.MultiWriter(hMd5, hSha1, hSha256)

	if _, err := io.Copy(multiWriter, f); err != nil {
		return nil, err
	}

	return &HashResults{
		MD5:    hex.EncodeToString(hMd5.Sum(nil)),
		SHA1:   hex.EncodeToString(hSha1.Sum(nil)),
		SHA256: hex.EncodeToString(hSha256.Sum(nil)),
	}, nil
}

// NewHashers returns initialized md5, sha1, sha256 hash.Hash objects
func NewHashers() (hash.Hash, hash.Hash, hash.Hash) {
	return md5.New(), sha1.New(), sha256.New()
}
