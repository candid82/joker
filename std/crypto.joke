(ns
  ^{:go-imports ["crypto/sha256" "crypto/sha512" "crypto/md5" "crypto/sha1"]
    :doc "Implements common cryptographic and hash functions."}
  crypto)

(defn ^String hmac
  "Returns HMAC signature for message and key using specified algorithm.
  Algorithm is one of the following: :sha1, :sha224, :sha256, :sha384, :sha512."
  {:added "1.0"
  :go "hmacSum(algorithm, message, key)"}
  [^Keyword algorithm ^String message ^String key])

(defn ^String sha256
  "Returns the SHA256 checksum of the data."
  {:added "1.0"
  :go "! t := sha256.Sum256([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha224
  "Returns the SHA224 checksum of the data."
  {:added "1.0"
  :go "! t := sha256.Sum224([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha384
  "Returns the SHA384 checksum of the data."
  {:added "1.0"
  :go "! t := sha512.Sum384([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha512
  "Returns the SHA512 checksum of the data."
  {:added "1.0"
  :go "! t := sha512.Sum512([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha512-224
  "Returns the SHA512/224 checksum of the data."
  {:added "1.0"
  :go "! t := sha512.Sum512_224([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha512-256
  "Returns the SHA512/256 checksum of the data."
  {:added "1.0"
  :go "! t := sha512.Sum512_256([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String md5
  "Returns the MD5 checksum of the data."
  {:added "1.0"
  :go "! t := md5.Sum([]byte(data)); _res := string(t[:])"}
  [^String data])

(defn ^String sha1
  "Returns the SHA1 checksum of the data."
  {:added "1.0"
  :go "! t := sha1.Sum([]byte(data)); _res := string(t[:])"}
  [^String data])
