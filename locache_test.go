package locache

import (
	"bytes"
	"encoding/gob"
	"os"
	"testing"
	"time"
)

var sampleData = []byte("very important data that needs to be cache")

type PersonTest struct {
	FirstName    string
	LastName     string
	Age          int
	Occupation   string
	Hobbies      []string
	FavoriteFood []string
	Bio          string
	DOB          time.Time
}

func TestLocache_Set(t *testing.T) {
	cache, teardown := setup(t, "./cache")
	defer teardown()

	if err := cache.Set("mydata", sampleData, 10); err != nil {
		t.Fatal(err)
	}
}

func TestLocache_Get(t *testing.T) {
	cache, teardown := setup(t, "./cache")
	defer teardown()

	p1 := PersonTest{
		FirstName:    "Bob",
		LastName:     "Builder",
		DOB:          time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		Age:          21,
		Occupation:   "Computer SciEntiSt",
		Hobbies:      []string{"Swimming", "Coding", "Playing Hockey", "Cycling"},
		FavoriteFood: []string{"Oxtail and yam with sweet potato", "mutton with white rice"},
		Bio:          "Hello my name is Bob and I love writing coding and cycling. I love being a scientist because it allows me to research and discover new things.Even though I no longer attend school, I have and will never stop learning, I love reading non-fictional books and watching educational tutorials in my free time.",
	}

	data, err := EncodeGob(p1)
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(string(data))

	if err := cache.Set("myCustomData", data, 10); err != nil {
		t.Fatal(err)
	}

	if cachedData, err := cache.Get("myCustomData"); err != nil {
		t.Fatal("could not retrieve data from cache: ", err)
	} else {
		//fmt.Println("\n", string(cachedData))
		var p2 PersonTest
		if err := DecodeGob(cachedData, &p2); err != nil {
			t.Fatal("could not decode result from cache: ", err)
		}
		if p1.FirstName != p2.FirstName || p1.LastName != p2.LastName || p2.Age != p2.Age || p1.Occupation != p2.Occupation || p1.Hobbies[0] != p2.Hobbies[0] || p1.Bio != p2.Bio {
			t.Fatal("got corrupted data from cache: ", cachedData)
		}
	}
}

func BenchmarkLocache_Set_Get(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache"})
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set("mydata", data, 1000); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
		if _, err := cache.Get("mydata"); err != nil {
			b.Fatal("could not retrieve data from cache")
		}
	}
}
func BenchmarkLocache_Get(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache"})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := cache.Get("mydata"); err != nil {
			b.Fatal("could not retrieve data from cache")
		}
	}
}
func BenchmarkLocache_Set(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache"})
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set("mydata", data, 2); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
	}
}

func setup(t *testing.T, directory string) (*Locache, func()) {
	teardown := func() {}
	cache, err := New(&Config{Directory: directory})
	if err != nil {
		t.Fatal(err)
	}

	teardown = func() {
		err = os.RemoveAll("./cache")
		if err != nil {
			t.Errorf("setup : RemoveAll:%v", err)
		}
	}

	return cache, teardown

}

func EncodeGob(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DecodeGob(b []byte, result interface{}) error {
	buf := bytes.NewBuffer(b)
	enc := gob.NewDecoder(buf)

	err := enc.Decode(result)
	if err != nil {
		return err
	}
	return nil

}
