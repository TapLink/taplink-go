package examples

import (
	"log"

	"github.com/bradberger/taplink-go"
)

func main() {

	api := taplink.New("my-api-key")
	pwd, err := api.NewPassword([]byte("my-password-hash"))
	if err != nil {
		log.Println("NewPassword error", err)
		return
	}

	verify, err := api.VerifyPassword([]byte("my-password-hash"), pwd.Hash, pwd.VersionID)
	if err != nil {
		log.Println("VerifyPassword error", err)
		return
	}

	log.Println("Did it match?", verify.Matched)
}
