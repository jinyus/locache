package locache

import (
	"fmt"
	"os"
	"testing"
	"time"
)

var sampleData = "very important data that needs to be cache"

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
	cache, teardown := setup(t, "./cache", 0)
	defer teardown()

	if err := cache.Set("mydata", &sampleData, 10); err != nil {
		t.Fatal(err)
	}
}

func TestLocache_Get(t *testing.T) {
	cache, teardown := setup(t, "./cache", 0)
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

	if err := cache.Set("myCustomData", &p1, time.Now().Add(5*time.Second).Unix()); err != nil {
		t.Fatal(err)
	}

	var p2 PersonTest
	if err := cache.Get("myCustomData", &p2); err != nil {
		t.Fatal("could not retrieve data from cache: ", err)
	} else {
		if p1.FirstName != p2.FirstName || p1.LastName != p2.LastName || p2.Age != p2.Age || p1.Occupation != p2.Occupation || p1.Hobbies[0] != p2.Hobbies[0] || p1.Bio != p2.Bio {
			t.Fatal("got corrupted data from cache")
		}
	}

	//GET key that doesn't exitst

	if err := cache.Get("nonexistentkey", &p2); err == nil {
		t.Fatal("an error should be returned for a key that doesn't exist: ", err)
	} else if errMsg := err.Error(); errMsg != keyDoesntExistsError {
		t.Fatalf("incorrect error returned for a key that doesn't exist:\nexpected : %s\nactual: %s \n", keyDoesntExistsError, errMsg)
	}

}

func TestJanitor(t *testing.T) {
	cache, _ := setup(t, "cache", 5*time.Second)
	expiryTimes := []time.Duration{
		2 * time.Second,
		800 * time.Millisecond,
		12 * time.Second,
		1300 * time.Millisecond,
	}

	for i := 0; i < 4; i++ {
		key := fmt.Sprint("item", i)
		if err := cache.Set(key, sampleData, time.Now().Add(expiryTimes[i]).Unix()); err != nil {
			t.Fatal(err)
		}
	}

	println("sleeping so janitor can clean up\n\n")
	time.Sleep(25 * time.Second)
	println("woke up\n")
}

func TestLocache_Get_Compress(t *testing.T) {
	cache, teardown := setupCompress(t, "./cache")
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

	if err := cache.Set("myCustomData", &p1, time.Now().Add(5*time.Second).Unix()); err != nil {
		t.Fatal(err)
	}

	var p2 PersonTest
	if err := cache.Get("myCustomData", &p2); err != nil {
		t.Fatal("could not retrieve data from cache: ", err)
	} else {
		if p1.FirstName != p2.FirstName || p1.LastName != p2.LastName || p2.Age != p2.Age || p1.Occupation != p2.Occupation || p1.Hobbies[0] != p2.Hobbies[0] || p1.Bio != p2.Bio {
			t.Fatal("got corrupted data from cach")
		}
	}
}

func BenchmarkLocache_Set_Get(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache", UseCompression: true})
	if err != nil {
		b.Fatal(err)
	}
	data := "very important data that needs to be cache"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set("mydata", &data, time.Now().Add(60*time.Second).Unix()); err != nil {
			b.Fatal("could not write data to cache: ", err)
		}
		var result string
		if err := cache.Get("mydata", &result); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
		if data != result {
			b.Fatal("wrong data received from cache : ", result)
		}
	}
}

func BenchmarkLocache_Get(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache", UseCompression: true})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var p2 PersonTest
		if err := cache.Get("mydata", &p2); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		} else {
			if p2.FirstName != "Bob" {
				b.Fatal("wrong first received from cache: ", p2.FirstName)
			}
		}
	}
}

func BenchmarkLocache_Set(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache", UseCompression: true})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
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

		if err := cache.Set("mydata", &p1, time.Now().Add(time.Hour).Unix()); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
	}
}

func setup(t *testing.T, directory string, cleanUpInterval time.Duration) (*Locache, func()) {
	teardown := func() {}
	cache, err := New(&Config{
		Directory:       directory,
		UseCompression:  true,
		CleanUpInterval: cleanUpInterval,
	})
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

func setupCompress(t *testing.T, directory string) (*Locache, func()) {
	teardown := func() {}
	cache, err := New(&Config{Directory: directory, UseCompression: true})
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
