package repository

const maxUserIDLen = 32

func validateUserID(userID string) error {
	if len(userID) > maxUserIDLen {
		return ErrUserIDLen
	}
	return nil
}
