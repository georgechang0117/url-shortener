package base62

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type base62TestSuite struct {
	suite.Suite
}

func (s *base62TestSuite) SetupSuite() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func TestBase62Suite(t *testing.T) {
	suite.Run(t, new(base62TestSuite))
}

func (s *base62TestSuite) TestEncodeAndDecode() {
	id := rand.Uint64()

	encoded := Encode(id)

	decoded, err := Decode(encoded)
	s.NoError(err)
	s.Equal(id, decoded)
}

func (s *base62TestSuite) TestDecodeInvalid() {
	encoded := "abc+e"

	_, err := Decode(encoded)
	s.Error(err)
}
