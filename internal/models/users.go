package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModel struct {
	DB *pgxpool.Pool
}

func (m *UserModel) Insert(name, email, password string) error {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created)
	VALUES($1, $2, $3, CURRENT_TIMESTAMP)`

	_, err = m.DB.Exec(context.Background(), stmt, name, email, string(hashedPassword))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && strings.Contains(pgErr.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
	}

	return nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {

	var id int
	var hashedPassword []byte

	stmt := "SELECT id, hashed_password FROM users WHERE email = $1"
	err := m.DB.QueryRow(context.Background(), stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	return id, nil

}

func (m *UserModel) Exists(id int) (bool, error) {
	return false, nil
}