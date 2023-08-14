package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	Get(id int) (*User, error)
	PasswordUpdate(id int, currentPassword, newPassword string) error
}

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

func (m *UserModel) Get(id int) (*User, error) {
	var user User
	stmt := `SELECT id, name, email, created FROM users WHERE id = $1`
	err := m.DB.QueryRow(context.Background(), stmt, id).Scan(&user.ID, &user.Name, &user.Email, &user.Created)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoRows
		} else {
			return nil, err
		}
	}
	return &user, nil
}

func (m *UserModel) Insert(name, email, password string) error {

	hashedPassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	stmt := `INSERT INTO users (name, email, hashed_password, created)
			VALUES($1, $2, $3, CURRENT_TIMESTAMP)`

	_, err = m.DB.Exec(context.Background(), stmt, name, email, hashedPassword)
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
	var hashedPassword string

	stmt := "SELECT id, hashed_password FROM users WHERE email = $1"
	err := m.DB.QueryRow(context.Background(), stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	if err != nil {
		return 0, err
	} else if !match {
		return 0, ErrInvalidCredentials
	}

	return id, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	var exists bool
	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = $1)"
	err := m.DB.QueryRow(context.Background(), stmt, id).Scan(&exists)
	return exists, err
}

func (m *UserModel) PasswordUpdate(id int, currentPassword, newPassword string) error {

	var currentHashedPassword string
	stmt := "SELECT hashed_password FROM users WHERE id = $1"
	err := m.DB.QueryRow(context.Background(), stmt, id).Scan(&currentHashedPassword)
	if err != nil {
		return err
	}

	match, err := argon2id.ComparePasswordAndHash(currentPassword, currentHashedPassword)
	if err != nil {
		return err
	} else if !match {
		return ErrInvalidCredentials
	}

	newHashedPassword, err := argon2id.CreateHash(newPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}
	stmt = "UPDATE users SET hashed_password = $1 WHERE id = $2"
	_, err = m.DB.Exec(context.Background(), stmt, newHashedPassword, id)
	return err
}
