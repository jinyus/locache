package locache

import (
	"bytes"
	"encoding/gob"
	"fmt"
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
	cache, _ := setup(t, "./cache", 0)
	//defer teardown()

	if err := cache.Set("mydata", sampleData, 10); err != nil {
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

	data, err := EncodeGob(p1)
	if err != nil {
		t.Fatal(err)
	}

	if err := cache.Set("myCustomData", data, 5*time.Second); err != nil {
		t.Fatal(err)
	}

	if cachedData, err := cache.Get("myCustomData"); err != nil {
		t.Fatal("could not retrieve data from cache: ", err)
	} else {
		//if bytes.Compare(cachedData, data) != 0 {
		//	t.Fatal("got corrupted data from cache. inputSize: ", len(testData), " cacheSize: ", len(cachedData))
		//}
		//return
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

func TestJanitor(t *testing.T) {
	cache, _ := setup(t, "cache", 5*time.Second)
	expiryTimes := []time.Duration{
		2 * time.Second,
		800 * time.Millisecond,
		6 * time.Second,
		1300 * time.Millisecond,
	}

	for i := 0; i < 4; i++ {
		key := fmt.Sprint("item", i)
		if err := cache.Set(key, sampleData, expiryTimes[i]); err != nil {
			t.Fatal(err)
		}
	}

	println("sleeping for get\n\n")
	time.Sleep(3 * time.Second)
	println("woke up to get\n")

	for i := 0; i < 4; i++ {
		key := fmt.Sprint("item", i)
		_, _ = cache.Get(key)
	}

	println("sleeping so janitor can clean up\n\n")
	time.Sleep(12 * time.Second)
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

	data, err := EncodeGob(p1)
	if err != nil {
		t.Fatal(err)
	}

	if err := cache.Set("myCustomData", data, 5*time.Second); err != nil {
		t.Fatal(err)
	}

	if cachedData, err := cache.Get("myCustomData"); err != nil {
		t.Fatal("could not retrieve data from cache: ", err)
	} else {
		//if bytes.Compare(cachedData, testData) != 0 {
		//	t.Fatal("got corrupted data from cache. inputSize: ", len(testData), " cacheSize: ", len(cachedData))
		//}
		//return
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
	cache, err := New(&Config{Directory: "./cache", UseCompression: true})
	if err != nil {
		b.Fatal(err)
	}
	data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set("mydata", data, time.Minute); err != nil {
			b.Fatal("could not write data to cache: ", err)
		}
		if _, err := cache.Get("mydata"); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
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
		if _, err := cache.Get("mydata"); err != nil {
			b.Fatal("could not retrieve data from cache: ", err)
		}
	}
}

func BenchmarkLocache_Set(b *testing.B) {
	cache, err := New(&Config{Directory: "./cache", UseCompression: true})
	if err != nil {
		b.Fatal(err)
	}
	//data := []byte("very important data that needs to be cache")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := cache.Set("mydata", testData, time.Hour); err != nil {
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

var testData = []byte(`{"Video":{"Title":"Big ass mom lets her virgin son fuck her!","Thumbnail":"httpss://cdn77-pic.xvideos-cdn.com/videos/thumbs169ll/df/78/3d/df783d78173e1b29949c7c8e9f5d6fdb/df783d78173e1b29949c7c8e9f5d6fdb.11.jpg","Keywords":["cum","teen","latina","ass","milf","butt","rough","amateur","homemade","fuck","french","mom","huge-ass","big-ass","family","son","phat","cul","pawg","perfect-ass"],"Rating":"4.3k","Mp4Url":"aHR0cHM6Ly9obHM0LWwfe2ba9d9zLnh2aWRlb3MtY2RuLmNvbS85NjA1MDdiOGYzYjUyMDk3ZjFiMmQzZGY0MThmZGEyYTQ5ZDM4YjNmLTE2MDA4MTk0NDQvdmlkZW9zL2hscy9kZi83OC8zZC9kZjc4M2Q3ODE3M2UxYjI5OTQ5YzdjOGU5ZjVkNmZkYi9obHMubTN1OA==","IsHLS":true},"VideoList":[{"id":58325963,"tf":"A young French lingerie model gets her big booty ass fucked!","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/61/39/25/613925d3af59ccd6fa6d434ec3053d23/613925d3af59ccd6fa6d434ec3053d23.21.jpg","u":"/porn/58325963/a-young-french-lingerie-model-gets-her-big-booty-ass-fucked-","d":"12 min","n":"24.9k"},{"id":58069611,"tf":"I love fuck a big ass teen in front of a reality show! French Amateur!","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/80/a5/f9/80a5f944c28817a420b9a9ecc8b68d16/80a5f944c28817a420b9a9ecc8b68d16.18.jpg","u":"/porn/58069611/i-love-fuck-a-big-ass-teen-in-front-of-a-reality-show-french-amateur-","d":"11 min","n":"132.3k"},{"id":42157537,"tf":"My dream fuck with ma","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/cd/9a/6f/cd9a6fb4b265cf8b13972adfc75621df/cd9a6fb4b265cf8b13972adfc75621df.7.jpg","u":"/porn/42157537/my-dream-fuck-with-ma","d":"42 sec","n":"1.3M"},{"id":56058049,"tf":"My step sister sucks me and lets me fuck her big ass during the work!","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/19/88/de/1988def97db4796744e0a514f60ca3f8/1988def97db4796744e0a514f60ca3f8.24.jpg","u":"/porn/56058049/my-step-sister-sucks-me-and-lets-me-fuck-her-big-ass-during-the-work-","d":"10 min","n":"3M"},{"id":57873903,"tf":"PURE TABOO 2 Step-Brothers DP Their Step-Mom","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/37/95/fe/3795fe5cd7c3fd72d0fddac7c7ef2942/3795fe5cd7c3fd72d0fddac7c7ef2942.14.jpg","u":"/porn/57873903/pure-taboo-2-step-brothers-dp-their-step-mom","d":"11 min","n":"3.4M"},{"id":57627505,"tf":"My Latina Mom catches me peeping on her","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/f0/91/59/f091598f2db22ae9457df1a2ec281857/f091598f2db22ae9457df1a2ec281857.7.jpg","u":"/porn/57627505/my-latina-mom-catches-me-peeping-on-her","d":"8 min","n":"1.1M"},{"id":43596321,"tf":"I love ass fuck","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/a8/39/97/a8399795be6886f9fa3e691c10d16dec/a8399795be6886f9fa3e691c10d16dec.22.jpg","u":"/porn/43596321/i-love-ass-fuck","d":"93 sec","n":"694.2k"},{"id":57958965,"tf":"MOM Amazing natural body babe Cherry Kiss has multiple squirting orgasms","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/f4/c6/ef/f4c6ef018c7a03f80ddca9c364b1e62e/f4c6ef018c7a03f80ddca9c364b1e62e.14.jpg","u":"/porn/57958965/mom-amazing-natural-body-babe-cherry-kiss-has-multiple-squirting-orgasms","d":"12 min","n":"191.8k"},{"id":58051495,"tf":"Teen gets a cock in her big ass in the garage","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/56/18/06/5618060791a4e664057f196a136b8784/5618060791a4e664057f196a136b8784.18.jpg","u":"/porn/58051495/teen-gets-a-cock-in-her-big-ass-in-the-garage","d":"12 min","n":"635.5k"},{"id":57303385,"tf":"French Model in sexy lingerie gets paid get fucked her big ass by the director!","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/50/48/81/504881cda298c014c06113021b98a337/504881cda298c014c06113021b98a337.2.jpg","u":"/porn/57303385/french-model-in-sexy-lingerie-gets-paid-get-fucked-her-big-ass-by-the-director-","d":"11 min","n":"565.4k"},{"id":50013391,"tf":"Big tits Latina mom lets stepson fuck her tight milf pussy and he cums all over her ass","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/2a/63/4e/2a634eb2207f2297594cd61798c6a100/2a634eb2207f2297594cd61798c6a100.23.jpg","u":"/porn/50013391/big-tits-latina-mom-lets-stepson-fuck-her-tight-milf-pussy-and-he-cums-all-over-her-ass","d":"7 min","n":"52.1k"},{"id":55072123,"tf":"Mom and son in a hotel","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/04/33/47/043347aa43ebc3aafc17dc887343b73d/043347aa43ebc3aafc17dc887343b73d.13.jpg","u":"/porn/55072123/mom-and-son-in-a-hotel","d":"10 min","n":"8M"},{"id":57798867,"tf":"Sexy red lingerie on this huge juicy ass French PAWG!","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/ad/9d/b3/ad9db3b7e5276e30ccba43b1d9775586/ad9db3b7e5276e30ccba43b1d9775586.24.jpg","u":"/porn/57798867/sexy-red-lingerie-on-this-huge-juicy-ass-french-pawg-","d":"14 min","n":"114.6k"},{"id":56080325,"tf":"Young But Mature Sexy Mom Gets Tied Up, Whipped, Gagged And Fucked Hard. Thick Ass/Bubble Butt/Chubby/BBW/Curvy/Plump/Chunky PAWG Gets Her Big Phat Round Ass Fucked Hard. Real Homemade Amateur Porn. Big Booty Mother Gets Force Fucked Hard \u0026amp; Loves It.","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/3f/77/5f/3f775f219822c5c171a80c94b63af66e/3f775f219822c5c171a80c94b63af66e.2.jpg","u":"/porn/56080325/young-but-mature-sexy-mom-gets-tied-up-whipped-gagged-and-fucked-hard.-thick-ass-bubble-butt-chubby-bbw-curvy-plump-chunky-pawg-gets-her-big-phat-round-ass-fucked-hard.-real-homemade-amateur-porn.-big-booty-mother-gets-force-fucked-hard-and-loves-it.","d":"5 min","n":"1.6M"},{"id":57967555,"tf":"Mi madre me iba provocando por el camino y al final no dud\u0026eacute; en follarme su rico culo. \u0026Uacute;nete a nuestro club de fans en www.onlyfans.com/ouset","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/c1/a6/67/c1a6678f9b64ca864f1dc658df8b1315/c1a6678f9b64ca864f1dc658df8b1315.29.jpg","u":"/porn/57967555/mi-madre-me-iba-provocando-por-el-camino-y-al-final-no-dude-en-follarme-su-rico-culo.-unete-a-nuestro-club-de-fans-en-www.onlyfans.com-ouset","d":"13 min","n":"1.2M"},{"id":58025647,"tf":"Drogu\u0026eacute; a mi t\u0026iacute;a y me folle su enorme culo mientras dorm\u0026iacute;a. \u0026Uacute;nete a nuestro club de fans en www.onlyfans.com/ouset","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/e2/c2/14/e2c214e492041d4525628d383ac7bcaf/e2c214e492041d4525628d383ac7bcaf.30.jpg","u":"/porn/58025647/drogue-a-mi-tia-y-me-folle-su-enorme-culo-mientras-dormia.-unete-a-nuestro-club-de-fans-en-www.onlyfans.com-ouset","d":"9 min","n":"4.6M"},{"id":54964767,"tf":"COMI DE VERDADE A M\u0026Atilde;E MEU AMIGO","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/21/8b/7f/218b7f161f4cd190aefdc13406956675/218b7f161f4cd190aefdc13406956675.25.jpg","u":"/porn/54964767/comi-de-verdade-a-mae-meu-amigo","d":"7 min","n":"5.9M"},{"id":53369101,"tf":"Nini Divine shows her huge ass in a sexy swimsuit, I fuck her big booty ! French Amateur !","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/7b/de/f9/7bdef9918024fa5d241df34bffe21884/7bdef9918024fa5d241df34bffe21884.9.jpg","u":"/porn/53369101/nini-divine-shows-her-huge-ass-in-a-sexy-swimsuit-i-fuck-her-big-booty-french-amateur-","d":"12 min","n":"760.2k"},{"id":55799877,"tf":"Mother \u0026amp; Son Practice Photography - Brianna Beach - Mom Comes First - Preview","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/6e/65/7a/6e657af05af480507aeeddfd256ddd19/6e657af05af480507aeeddfd256ddd19.22.jpg","u":"/porn/55799877/mother-and-son-practice-photography---brianna-beach---mom-comes-first---preview","d":"10 min","n":"7.7M"},{"id":55777309,"tf":"Siblings fuck on camera","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/c8/44/98/c84498cf5010aeab8ebdb71ae87532bd/c84498cf5010aeab8ebdb71ae87532bd.17.jpg","u":"/porn/55777309/siblings-fuck-on-camera","d":"7 min","n":"2.1M"},{"id":55073543,"tf":"That dick in there","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/7d/cd/0a/7dcd0aac64bb3099bdd4242672f899fb/7dcd0aac64bb3099bdd4242672f899fb.30.jpg","u":"/porn/55073543/that-dick-in-there","d":"2 min","n":"353.4k"},{"id":57797363,"tf":"Perverse Family 1","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/62/05/20/620520d44da82472a17ca68169b012ab/620520d44da82472a17ca68169b012ab.13.jpg","u":"/porn/57797363/perverse-family-1","d":"17 min","n":"569.7k"},{"id":57852367,"tf":"Son fingers his sleeping mommy - fucked up family sex","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/1e/12/62/1e12629897d02508992c614f51d75e2c/1e12629897d02508992c614f51d75e2c.3.jpg","u":"/porn/57852367/son-fingers-his-sleeping-mommy---fucked-up-family-sex","d":"8 min","n":"331.3k"},{"id":55590077,"tf":"He cums twice on my huge ass! French Amateur Couple!","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/f1/03/56/f10356af81cfa92ea3260da0e7012fc5/f10356af81cfa92ea3260da0e7012fc5.17.jpg","u":"/porn/55590077/he-cums-twice-on-my-huge-ass-french-amateur-couple-","d":"9 min","n":"679.9k"},{"id":53785563,"tf":"Sleeping Mom Fucks Blows \u0026amp; Fucks Horny Son- Nina Kayy","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/28/0f/6b/280f6bedfce7cc72a1e881dfc7fa3fd5/280f6bedfce7cc72a1e881dfc7fa3fd5.8.jpg","u":"/porn/53785563/sleeping-mom-fucks-blows-and-fucks-horny-son--nina-kayy","d":"8 min","n":"999.5k"},{"id":55827001,"tf":"a big butt ebony gets ready for the fuck table","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/aa/24/34/aa2434f6ca4a38bc976da826d32e42f8/aa2434f6ca4a38bc976da826d32e42f8.20.jpg","u":"/porn/55827001/a-big-butt-ebony-gets-ready-for-the-fuck-table","d":"3 min","n":"6.5M"},{"id":57499707,"tf":"Mommy, We Want To Fuck You! - Syren De Mer","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/df/f6/22/dff6225e73116ce21ae12b4e1af0debb/dff6225e73116ce21ae12b4e1af0debb.15.jpg","u":"/porn/57499707/mommy-we-want-to-fuck-you---syren-de-mer","d":"6 min","n":"414.9k"},{"id":36116409,"tf":"Young Boy Fuck Sexy Mom","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/b1/9c/5c/b19c5cec0bb8c3978341701502020fb0/b19c5cec0bb8c3978341701502020fb0.14.jpg","u":"/porn/36116409/young-boy-fuck-sexy-mom","d":"9 min","n":"1.9M"},{"id":51947529,"tf":"Hot Step Mom Finds Dirty Deeds of Her Stepson | Stepmom and Son","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/bc/a3/68/bca3683f885b918573ea63e17d1d4dc9/bca3683f885b918573ea63e17d1d4dc9.30.jpg","u":"/porn/51947529/hot-step-mom-finds-dirty-deeds-of-her-stepson-stepmom-and-son","d":"8 min","n":"4M"},{"id":57727967,"tf":"Busty Thick Daughter Fucks Daddy In Front Of Mom- Cara May","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/25/35/b1/2535b16cea04838d2336bb36ce14b88a/2535b16cea04838d2336bb36ce14b88a.17.jpg","u":"/porn/57727967/busty-thick-daughter-fucks-daddy-in-front-of-mom--cara-may","d":"8 min","n":"414.4k"},{"id":22489397,"tf":"Cheating Wife Sucks Black Cock From Internet","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/98/ce/41/98ce41b81616d9ee93bd86f6785f0bd3/98ce41b81616d9ee93bd86f6785f0bd3.16.jpg","u":"/porn/22489397/cheating-wife-sucks-black-cock-from-internet","d":"8 min","n":"2.2M"},{"id":37916739,"tf":"Mom And Son Share A Hotel Room","i":"https://cdn77-pic.xvideos-cdn.com/videos/thumbs169/74/22/66/74226672f757677e4191d09715380122/74226672f757677e4191d09715380122.16.jpg","u":"/porn/37916739/mom-and-son-share-a-hotel-room","d":"16 min","n":"20.3M"},{"id":57741081,"tf":"Big Ass Big Tits Latina MILF Stepmom Miss Raquel Lets Young Stepson Fuck Her On Family Kitchen Table","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/ce/87/ad/ce87ad465fb06722875e545d7e55370c/ce87ad465fb06722875e545d7e55370c.30.jpg","u":"/porn/57741081/big-ass-big-tits-latina-milf-stepmom-miss-raquel-lets-young-stepson-fuck-her-on-family-kitchen-table","d":"8 min","n":"525.2k"},{"id":56609849,"tf":"A big ass French Milf gets fuck by her gardener in her basement!","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/8a/33/49/8a334954a0bada31a6192314d26e70cb/8a334954a0bada31a6192314d26e70cb.22.jpg","u":"/porn/56609849/a-big-ass-french-milf-gets-fuck-by-her-gardener-in-her-basement-","d":"12 min","n":"666.8k"},{"id":34221473,"tf":"Black babe fucked rough","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/d6/c3/2c/d6c32cad1ff00644067f44238f9e8085/d6c32cad1ff00644067f44238f9e8085.3.jpg","u":"/porn/34221473/black-babe-fucked-rough","d":"10 min","n":"906.7k"},{"id":22608429,"tf":"notre musique","i":"https://img-l3.xvideos-cdn.com/videos/thumbs169/c1/32/b1/c132b1d2b6e23eeb55fffd2e150def5d/c132b1d2b6e23eeb55fffd2e150def5d.15.jpg","u":"/porn/22608429/notre-musique","d":"11 min","n":"37.9M"},{"id":57570215,"tf":"Son Wants To Force Fuck Stuck Mom First- Becky Bandini","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/2e/b0/21/2eb02183a1d27b3913179a5b1eb01ffd/2eb02183a1d27b3913179a5b1eb01ffd.22.jpg","u":"/porn/57570215/son-wants-to-force-fuck-stuck-mom-first--becky-bandini","d":"8 min","n":"2.2M"},{"id":54156009,"tf":"Loud Moaning African Girl fucked Brother BBC.","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/96/77/c7/9677c78c6537f8e5e42f55bc1f8f6048/9677c78c6537f8e5e42f55bc1f8f6048.9.jpg","u":"/porn/54156009/loud-moaning-african-girl-fucked-brother-bbc.","d":"6 min","n":"1.2M"},{"id":16684355,"tf":"That Ass Tho 7","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/6b/bf/68/6bbf6828c7613c7035740c0c05a0e46d/6bbf6828c7613c7035740c0c05a0e46d.19.jpg","u":"/porn/16684355/that-ass-tho-7","d":"7 min","n":"2.5M"},{"id":44247187,"tf":"Mom son sex","i":"https://img-hw.xvideos-cdn.com/videos/thumbs169/6c/c2/33/6cc233e0b45c94340511ac03afb31e5f/6cc233e0b45c94340511ac03afb31e5f.30.jpg","u":"/porn/44247187/mom-son-sex","d":"37 sec","n":"8.3M"}],"CamList":[{"thumb":"https://i1.wp.com/media.camsoda.com/thumbs/1800/jessiejoyce.jpg?lb=360,270","viewers":124,"username":"jessiejoyce","path":"/cams/c/jessiejoyce","provider":"camsoda"},{"thumb":"https://roomimg.stream.highwebmedia.com/ri/oliviaowens.jpg","viewers":10634,"username":"oliviaowens","path":"https://live.camservants.com/oliviaowens/?track=mfblive","provider":"camservants"},{"thumb":"https://i2.wp.com/media.camsoda.com/thumbs/1514/ivymmiller.jpg?lb=360,270","viewers":80,"username":"ivymmiller","path":"/cams/c/ivymmiller","provider":"camsoda"},{"thumb":"https://i0.wp.com/widgets.stripst.com/eu4/previews/1600672160/26267537?resize=360,270","viewers":1275,"username":"BabesGoWild","path":"/cams/s/BabesGoWild","provider":"stripchat"}],"GenderCams":null,"NextPage":0,"PrevPage":0,"Keyword":"","KeywordNoSp":"","Url":"https://www.myfreeblack.com/porn/58130029/","IsMobile":false,"HasMorePages":false}`)
