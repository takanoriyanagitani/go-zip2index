package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	zi "github.com/takanoriyanagitani/go-zip2index"
	. "github.com/takanoriyanagitani/go-zip2index/util"
)

var envValByKey func(string) IO[string] = Lift(
	func(key string) (string, error) {
		val, found := os.LookupEnv(key)
		switch found {
		case true:
			return val, nil
		default:
			return "", fmt.Errorf("env var %s missing", key)
		}
	},
)

var filename IO[string] = envValByKey("ENV_ZIP_FILENAME")

var derBytes IO[[]byte] = Bind(
	filename,
	Lift(zi.ZipFilenameToBasicInfoDerBytes),
)

func bytes2writer(wtr io.Writer) func([]byte) IO[Void] {
	return Lift(func(dat []byte) (Void, error) {
		_, e := wtr.Write(dat)
		return Empty, e
	})
}

var bytes2stdout func([]byte) IO[Void] = bytes2writer(os.Stdout)

var zfile2info2der2stdout IO[Void] = Bind(derBytes, bytes2stdout)

func main() {
	_, e := zfile2info2der2stdout(context.Background())
	if nil != e {
		log.Printf("%v\n", e)
	}
}
