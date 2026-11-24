// Rotates CRYPTO_KEY: re-encrypts everything the app stores encrypted from the
// current key to a new one. See docs/key-rotation.md for the full procedure.
//
//	go run cmd/rotatekey/rotatekey.go gen            # print a fresh key (run it yourself)
//	go run cmd/rotatekey/rotatekey.go check          # decrypt everything with CRYPTO_KEY, change nothing
//	go run cmd/rotatekey/rotatekey.go rotate         # dry run: do it all, then roll back
//	go run cmd/rotatekey/rotatekey.go rotate -apply  # really do it (one transaction)
//
// The current key is CRYPTO_KEY and the new one NEW_CRYPTO_KEY, both from the
// environment or .env. Neither is ever printed (except by gen, which prints
// only the new random key). Stop the app before an -apply run.
package main

import (
	"fmt"
	"os"

	"journal/internal/database"
	"journal/internal/keyrotation"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "gen":
		k, err := keyrotation.GenerateKey()
		if err != nil {
			fail(err)
		}
		fmt.Println(k)
	case "check":
		key := []byte(os.Getenv("CRYPTO_KEY"))
		if os.Getenv("NEW_CRYPTO_KEY") != "" && len(os.Args) > 2 && os.Args[2] == "-new" {
			key = []byte(os.Getenv("NEW_CRYPTO_KEY"))
		}
		rep, err := keyrotation.Check(database.New().DB, key)
		print(rep)
		if err != nil {
			fail(err)
		}
		fmt.Println("OK: every encrypted value decrypts with the key.")
	case "rotate":
		apply := len(os.Args) > 2 && os.Args[2] == "-apply"
		if os.Getenv("APP_ENV") == "production" && apply && os.Getenv("CONFIRM_ROTATE") != "yes" {
			fail(fmt.Errorf("APP_ENV is production: set CONFIRM_ROTATE=yes to apply (stop the app first)"))
		}
		rep, err := keyrotation.Rotate(database.New().DB,
			[]byte(os.Getenv("CRYPTO_KEY")), []byte(os.Getenv("NEW_CRYPTO_KEY")), apply)
		print(rep)
		if err != nil {
			fail(err)
		}
		if apply {
			fmt.Printf("DONE: re-encrypted %d values. Now set CRYPTO_KEY to the new key and start the app.\n", rep.Total())
		} else {
			fmt.Printf("DRY RUN OK: %d values would be re-encrypted. Nothing was changed.\n", rep.Total())
		}
	default:
		usage()
	}
}

func print(rep keyrotation.Report) {
	for _, t := range rep.Tables {
		fmt.Printf("%-24s rows=%-5d encrypted=%-5d plaintext=%-3d empty=%d\n", t.Table, t.Rows, t.Encrypted, t.Plaintext, t.Empty)
	}
}

func usage() {
	fmt.Println("usage: rotatekey gen | check [-new] | rotate [-apply]")
	os.Exit(2)
}

func fail(err error) {
	fmt.Println("ERROR:", err)
	os.Exit(1)
}
