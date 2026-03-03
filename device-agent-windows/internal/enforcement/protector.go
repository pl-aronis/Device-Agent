package enforcement

import (
	"errors"
	"log"
	"regexp"
	"strings"
)

func (b *Bitlocker) addRecoveryPassword() (string, string, error) {

	out, err := b.exec.Run(manageBDE, "-protectors", "-add", "C:", "-RecoveryPassword")
	if err != nil {
		return "", "", err
	}

	keyRe := regexp.MustCompile(`(\d{6}-){7}\d{6}`)
	idRe := regexp.MustCompile(`ID:\s*{[^}]+}`)

	key := keyRe.FindString(out)
	idMatch := idRe.FindString(out)

	if key == "" || idMatch == "" {
		return "", "", errors.New("failed to extract recovery key or ID")
	}

	id := strings.TrimPrefix(idMatch, "ID: ")

	return key, id, nil
}

func (b *Bitlocker) deleteProtectorByID(id string) {
	b.exec.Run(manageBDE, "-protectors", "-delete", "C:", "-id", id)
}

func DeleteProtector(id string) {
	execImpl := &CommandExecutor{}
	execImpl.Run(manageBDE, "-protectors", "-delete", "C:", "-id", id)
}

func (b *Bitlocker) logRecoveryKey(key string) {
	log.Println("==============================================")
	log.Println("[RECOVERY PASSWORD]")
	log.Println(key)
	log.Println("==============================================")
}
