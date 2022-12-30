package staticcache

// byteView holds an immutable view of bytes.
type byteView struct {
    bs []byte
}

func (bv byteView) Len() int {
    return len(bv.bs)
}

func (bv byteView) String() string {
    return string(bv.bs)
}

func (bv byteView) ByteSlice() []byte {
    return cloneBytes(bv.bs)
}

func cloneBytes(bs []byte) []byte {
    copied := make([]byte, len(bs))
    copy(copied, bs)
    return copied
}
