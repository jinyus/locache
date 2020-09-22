package locache

import (
	"os"
	"testing"
)

func TestLocache(t *testing.T) {
	cache, _ := setup(t, "./cache")
	//defer teardown()
	data := []byte("very important data that needs to be cache")
	cache.Set("mydata", data, 1000)

	if cachedData, err := cache.Get("mydata"); err != nil {
		t.Fatal("could not retrieve data from cache")
	} else {
		if string(cachedData) != string(data) {
			t.Fatal("got corrupted data from cache: ", string(cachedData))
		}
	}

}

func TestLocache2(t *testing.T) {
	cache, _ := setup(t, "./cache2")
	//defer teardown()
	data := []byte("very important data that needs to be cache")
	err := cache.Set2("mydata2", data, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if cachedData, ok := cache.Get2("mydata2"); !ok {
		t.Fatal("could not retrieve data from cache")
	} else {
		if string(cachedData) != string(data) {
			t.Fatal("got corrupted data from cache: ", string(cachedData))
		}
	}

}

func BenchmarkLocache(b *testing.B) {
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

func BenchmarkLocache2(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache2"})
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set2("mydata", data, 1000); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
		if _, ok := cache.Get2("mydata"); !ok {
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

func BenchmarkLocache_Get2(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache2"})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, ok := cache.Get2("mydata2"); !ok {
			b.Fatal("could not retrieve data from cache")
		}
	}
}

func BenchmarkLocache_Set2(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache2"})
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set2("mydata2", data, 2); err != nil {
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
