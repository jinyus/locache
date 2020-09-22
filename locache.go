package locache

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/GitbookIO/syncgroup"
)

const (
	keyDoesntExistsError = "locache : key doesn't exists"
	ttlParsingError      = "locache : error parsing first line of cache file"
	keyExpiredError      = "locache : key has expired"
)

type Locache struct {
	directory string
	lock      *syncgroup.MutexGroup
}

type Config struct {
	Directory string
}

func New(cfg *Config) (*Locache, error) {
	// Create Locache directory
	if err := os.MkdirAll(cfg.Directory, os.ModePerm); err != nil {
		return nil, err
	}

	// Create Locache instance
	c := &Locache{
		directory: cfg.Directory,
		lock:      syncgroup.NewMutexGroup(),
	}

	return c, nil
}

//Caches the data provided and sets the expiration date
//based on the Time To Live(TTL) provided
func (c *Locache) Set(key string, data []byte, TTL time.Duration) error {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	// Open file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	expiryDate := time.Now().Add(TTL).Unix()
	expiryDateFmt := []byte(fmt.Sprintf("%d\n", expiryDate))

	// Write expDate on the first line of the file
	if _, err := file.Write(expiryDateFmt); err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}

//encode expiry with data as cacheitem struct
//func (c *Locache) Set2(key string, data []byte, timeoutInSeconds int64) error {
//	// Get encoded key
//	filename := c.getFilename(key)
//
//	// Lock for writing
//	c.lock.Lock(filename)
//	defer c.lock.Unlock(filename)
//
//	// Open file
//	file, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//
//	expiryDate := time.Now().Add(time.Second * time.Duration(timeoutInSeconds)).Unix()
//	item := CacheItem{
//		Data:   data,
//		Expiry: expiryDate,
//	}
//	encodedData, err := EncodeGob(item)
//	if err != nil {
//		return err
//	}
//	// Write data
//	_, err = file.Write(encodedData)
//	return err
//}

func (c *Locache) Delete(key string) error {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for writing
	c.lock.Lock(filename)
	defer c.lock.Unlock(filename)

	if ok, err := exists(filename); ok {
		return os.Remove(filename)
	} else {
		return err
	}
}

//Looks up the key in the cache,
//keyExpiredError will be returned if the data has expired
func (c *Locache) Get(key string) ([]byte, error) {
	// Get encoded key
	filename := c.getFilename(key)

	// Lock for reading
	c.lock.RLock(filename)
	defer c.lock.RUnlock(filename)

	if ok, err := exists(filename); !ok || err != nil {
		return nil, fmt.Errorf("%s: %v", keyDoesntExistsError, err)
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data []byte
	var count int

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if count == 0 {
			//reads first line to get expiry date
			count++
			txt := scanner.Text()
			ttl, err := strconv.Atoi(txt)
			if err != nil {
				return nil, fmt.Errorf("%s: %v", ttlParsingError, err)
			}
			if int64(ttl) < time.Now().Unix() {
				return nil, errors.New(keyExpiredError)
			}
			continue
		}
		data = append(data, []byte(scanner.Text()+"\n")...)
	}

	return data, nil
}

//func (c *Locache) Get2(key string) ([]byte, bool) {
//	// Get encoded key
//	filename := c.getFilename(key)
//
//	// Lock for reading
//	c.lock.RLock(filename)
//	defer c.lock.RUnlock(filename)
//
//	// Open file
//	file, err := os.Open(filename)
//	if err != nil {
//		return nil, false
//	}
//	defer file.Close()
//
//	// Read file
//	data, err := ioutil.ReadAll(file)
//	if err != nil {
//		log.Printf("Locache: Error reading from file %s\n", key)
//		return nil, false
//	}
//
//	var temp CacheItem
//
//	err = DecodeGob(data, &temp)
//	if err != nil {
//		fmt.Println("could not decode file content: ", err)
//		return nil, false
//	}
//
//	if temp.Expiry < time.Now().Unix() {
//		//defer func() { c.Delete(key) }()
//		//fmt.Println(key, " has expired")
//		return nil, false
//	}
//	return temp.Data, true
//}

func (c *Locache) Clean() error {
	// Delete directory
	if err := os.RemoveAll(c.directory); err != nil {
		return err
	}
	// Recreate directory
	return os.MkdirAll(c.directory, os.ModePerm)
}

func (c *Locache) getFilename(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return path.Join(c.directory, hex.EncodeToString(hasher.Sum(nil)))
}

// check file exist.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
