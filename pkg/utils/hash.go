package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"

	"github.com/massonsky/buffalo/pkg/errors"
)

// HashAlgorithm represents supported hash algorithms.
type HashAlgorithm string

const (
	MD5    HashAlgorithm = "md5"
	SHA1   HashAlgorithm = "sha1"
	SHA256 HashAlgorithm = "sha256"
	SHA512 HashAlgorithm = "sha512"
)

// ComputeHash computes the hash of data using the specified algorithm.
func ComputeHash(data []byte, algo HashAlgorithm) (string, error) {
	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}

	if _, err := h.Write(data); err != nil {
		return "", errors.Wrap(err, errors.ErrInternal, "failed to compute hash")
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeFileHash computes the hash of a file using the specified algorithm.
func ComputeFileHash(path string, algo HashAlgorithm) (string, error) {
	if path == "" {
		return "", errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.Wrap(err, errors.ErrNotFound, "file not found: %s", path)
		}
		return "", errors.Wrap(err, errors.ErrIO, "failed to open file: %s", path)
	}
	defer file.Close()

	h, err := newHasher(algo)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", errors.Wrap(err, errors.ErrIO, "failed to read file for hashing: %s", path)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeStringHash computes the hash of a string using the specified algorithm.
func ComputeStringHash(s string, algo HashAlgorithm) (string, error) {
	return ComputeHash([]byte(s), algo)
}

// VerifyFileHash verifies that a file's hash matches the expected hash.
func VerifyFileHash(path, expectedHash string, algo HashAlgorithm) (bool, error) {
	actualHash, err := ComputeFileHash(path, algo)
	if err != nil {
		return false, err
	}

	return actualHash == expectedHash, nil
}

// newHasher creates a new hasher for the specified algorithm.
func newHasher(algo HashAlgorithm) (hash.Hash, error) {
	switch algo {
	case MD5:
		return md5.New(), nil
	case SHA1:
		return sha1.New(), nil
	case SHA256:
		return sha256.New(), nil
	case SHA512:
		return sha512.New(), nil
	default:
		return nil, errors.New(errors.ErrInvalidArgument, "unsupported hash algorithm: %s", algo)
	}
}

// HashFile is a convenience function that uses SHA256 by default.
func HashFile(path string) (string, error) {
	return ComputeFileHash(path, SHA256)
}

// HashString is a convenience function that uses SHA256 by default.
func HashString(s string) (string, error) {
	return ComputeStringHash(s, SHA256)
}

// HashData is a convenience function that uses SHA256 by default.
func HashData(data []byte) (string, error) {
	return ComputeHash(data, SHA256)
}
