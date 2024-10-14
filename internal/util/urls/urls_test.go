package urls_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"
)

func TestResolveResourceName(t *testing.T) {
	resUrl := `https://ccp-bj29-video-preview.oss-enet.aliyuncs.com/lt/34860EF2932C94202A63D4F29C188097043205F9_1300259473__sha1_bj29/subtitle/subtitle_3.vtt?di=bj29&dr=339781490&f=64617b3bae2de1e1a8df40898a3952be207aca70&pds-params=%7B%22ap%22%3A%2276917ccccd4441c39457a04f6084fb2f%22%7D&security-token=CAISvgJ1q6Ft5B2yfSjIr5fdCNPapqtx8qqBRFT3pzZnerYegf3uoDz2IHhMf3NpBOkZvvQ1lGlU6%2Fcalq5rR4QAXlDfNQOaZ3ueq1HPWZHInuDox55m4cTXNAr%2BIhr%2F29CoEIedZdjBe%2FCrRknZnytou9XTfimjWFrXWv%2Fgy%2BQQDLItUxK%2FcCBNCfpPOwJms7V6D3bKMuu3OROY6Qi5TmgQ41Uh1jgjtPzkkpfFtkGF1GeXkLFF%2B97DRbG%2FdNRpMZtFVNO44fd7bKKp0lQLs0ARrv4r1fMUqW2X543AUgFLhy2KKMPY99xpFgh9a7j0iCbSGyUu%2FhcRm5sw9%2Byfo34lVYneY7xZ%2ByHN7uHwufJ7FxfIREfquk63pvSlHLcLPe0Kjzzleo2k1XRPVFF%2B535IaHXuToXDnvSi14GOAfXtuMkagAFOD20a2BT1Wf4wXbyRcR0HqWAtw6i4kBO%2FKsslS04SG6AUnRimmPPJrKlvqjGheg3hUwe%2Bky9jH8AJ2d9zU0Og9msrSSOY%2FEgqydcHEFhYcwDhXIQbA7Iyt18mqoFDkBrYwe0NSB5bm%2BlDCUbi2L68sXFkAD7HKKS1Z%2FKCFYrn9SAA&u=6780dc8ea26d48ac88981c851052d77c&x-oss-access-key-id=STS.NThCinKtPEhjFrFC62v92n8EB&x-oss-expires=1726738969&x-oss-signature=DWIp%2FJW0CPl4kennw6n6yAaKVCGrazUBXE5qho%2Ba8xk%3D&x-oss-signature-version=OSS2`
	fmt.Printf("urls.ResolveResourceName(resUrl): %v\n", urls.ResolveResourceName(resUrl))
}

func TestAppendUrlArgs(t *testing.T) {
	rawUrl := "http://localhost:8095/emby/Items/2008/PlaybackInfo?reqformat=json"
	res := urls.AppendArgs(rawUrl, "ambitious", "jun", "Static", "true", "unvalid")
	log.Println("拼接后的结果: ", res)
}

func TestIsRemote(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "rtp", args: args{path: "rtp://1.2.3.4:9999"}, want: true},
		{name: "http", args: args{path: "http://localhost:8095/emby/videos/53507/stream"}, want: true},
		{name: "https", args: args{path: "https://localhost:8095/emby/videos/53507/stream"}, want: true},
		{name: "file-unix", args: args{path: "/usr/local/app/test.mp4"}, want: false},
		{name: "file-windows", args: args{path: `D:\user\local\app\test.mp4`}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := urls.IsRemote(tt.args.path); got != tt.want {
				t.Errorf("IsRemote() = %v, want %v", got, tt.want)
			}
		})
	}
}
