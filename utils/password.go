package utils

import "golang.org/x/crypto/bcrypt"



// VerifyPassword compares a bcrypt hashed password with its plain-text version
func VerifyPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
