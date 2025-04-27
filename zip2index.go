package zip2index

import (
	"archive/zip"
	"encoding/asn1"
	"io"
	"iter"
	"os"
)

type CompressionMethod = asn1.Enumerated

const (
	CompressionMethodUnspecified CompressionMethod = 0
	CompressionMethodStore       CompressionMethod = 100
	CompressionMethodDeflate     CompressionMethod = 108
)

type ZipMethod uint16

func (m ZipMethod) ToCompressionMethod() CompressionMethod {
	switch uint16(m) {
	case zip.Store:
		return CompressionMethodStore
	case zip.Deflate:
		return CompressionMethodDeflate
	default:
		return CompressionMethodUnspecified
	}
}

type UnixtimeUs int64

type Checksum int64

type CompressedSize int64

type OriginalSize int64

// The offset value relative to the beginning of the zip file.
type Offset int64

type BasicZipItemInfo struct {
	Name     string `asn1:"utf8"`
	Modified UnixtimeUs
	Offset
	CompressedSize
	OriginalSize
	Checksum
	CompressionMethod
}

func (b BasicZipItemInfo) WithOffset(o Offset) BasicZipItemInfo {
	b.Offset = o
	return b
}

type ZipFileHeader zip.FileHeader

func (h ZipFileHeader) toBasicInfo() BasicZipItemInfo {
	return BasicZipItemInfo{
		Name:              h.Name,
		Modified:          UnixtimeUs(h.Modified.UnixMicro()),
		CompressionMethod: h.ToMethod(),
		Checksum:          Checksum(int64(h.CRC32)),
		CompressedSize:    CompressedSize(int64(h.CompressedSize64)),
		OriginalSize:      OriginalSize(int64(h.UncompressedSize64)),
		Offset:            0,
	}
}

func (h ZipFileHeader) ToMethod() CompressionMethod {
	var method uint16 = h.Method
	return ZipMethod(method).ToCompressionMethod()
}

func (h ZipFileHeader) ToBasicInfo(o Offset) BasicZipItemInfo {
	return h.toBasicInfo().WithOffset(o)
}

type ZipItem struct{ *zip.File }

func (i ZipItem) ToOffset() (Offset, error) {
	o, e := i.File.DataOffset()
	return Offset(o), e
}

func (i ZipItem) ToBasicInfo() (BasicZipItemInfo, error) {
	o, e := i.ToOffset()
	return ZipFileHeader(i.File.FileHeader).ToBasicInfo(o), e
}

type ZipArchive struct{ *zip.Reader }

func (a ZipArchive) Files() []*zip.File { return a.Reader.File }

func (a ZipArchive) ToBasicInfo() iter.Seq2[BasicZipItemInfo, error] {
	return func(yield func(BasicZipItemInfo, error) bool) {
		var files []*zip.File = a.Files()
		for _, item := range files {
			zitem := ZipItem{File: item}
			b, e := zitem.ToBasicInfo()
			if !yield(b, e) {
				return
			}
		}
	}
}

func (a ZipArchive) ToBasicInfoDerBytes() ([]byte, error) {
	var it iter.Seq2[BasicZipItemInfo, error] = a.ToBasicInfo()
	return BasicZipItemIter(it).ToDerBytes()
}

type ZipFileLike struct {
	io.ReaderAt
	Size int64
}

func (l ZipFileLike) ToArchive() (ZipArchive, error) {
	rdr, e := zip.NewReader(l.ReaderAt, l.Size)
	return ZipArchive{Reader: rdr}, e
}

func (l ZipFileLike) ToBasicInfoDerBytes() ([]byte, error) {
	a, e := l.ToArchive()
	if nil != e {
		return nil, e
	}
	return a.ToBasicInfoDerBytes()
}

type FileOs struct{ *os.File }

func (o FileOs) Size() (int64, error) {
	stat, e := o.File.Stat()
	if nil != e {
		return 0, e
	}
	return stat.Size(), nil
}

func (o FileOs) ToBasicInfoDerBytes() ([]byte, error) {
	size, e := o.Size()
	if nil != e {
		return nil, e
	}
	return ZipFileLike{ReaderAt: o.File, Size: size}.ToBasicInfoDerBytes()
}

func ZipFilenameToBasicInfoDerBytes(filename string) ([]byte, error) {
	f, e := os.Open(filename)
	if nil != e {
		return nil, e
	}
	defer f.Close()
	return FileOs{File: f}.ToBasicInfoDerBytes()
}

type BasicZipItemIter iter.Seq2[BasicZipItemInfo, error]

func (i BasicZipItemIter) Collect() ([]BasicZipItemInfo, error) {
	var ret []BasicZipItemInfo
	for b, e := range i {
		if nil != e {
			return nil, e
		}
		ret = append(ret, b)
	}
	return ret, nil
}

func (i BasicZipItemIter) ToDerBytes() ([]byte, error) {
	index, e := i.Collect()
	if nil != e {
		return nil, e
	}
	return BasicZipIndexInfo(index).ToDerBytes()
}

type BasicZipIndexInfo []BasicZipItemInfo

func (s BasicZipIndexInfo) ToDerBytes() ([]byte, error) {
	return asn1.Marshal(s)
}
