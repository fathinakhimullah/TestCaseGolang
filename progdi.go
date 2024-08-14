package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func initDB() {
	var err error
	// Sesuaikan konfigurasi koneksi database di sini
	db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/test_case")
	if err != nil {
		log.Fatal(err)
	}
}

func registerUser(c *gin.Context) {
	var user struct {
		Namauser     string `json:"namauser"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		Isactive     int    `json:"isactive"`
		NoCif        string `json:"nocif"`
		Alamatdetail string `json:"alamatdetail"`
		Provinsi     string `json:"provinsi"`
		Kabupaten    string `json:"kabupaten"`
		Kodepos      string `json:"kodepos"`
	}

	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert into Tbluser
	res, err := db.Exec("INSERT INTO Tbluser (Namauser, Email, Password, Isactive, noCif) VALUES (?, ?, ?, ?, ?)",
		user.Namauser, user.Email, user.Password, user.Isactive, user.NoCif)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	userID, err := res.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user ID"})
		return
	}

	// Insert into Alamatuser
	_, err = db.Exec("INSERT INTO Alamatuser (Iduser, Alamatdetail, Provinsi, Kabupaten, Kodepos) VALUES (?, ?, ?, ?, ?)",
		userID, user.Alamatdetail, user.Provinsi, user.Kabupaten, user.Kodepos)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func loginUser(c *gin.Context) {
	var login struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BindJSON(&login); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userId int
	var isLogin int
	var expiredlock sql.NullTime

	// Cek apakah email ada
	err := db.QueryRow("SELECT Iduser FROM Tbluser WHERE Email = ?", login.Email).Scan(&userId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Cek apakah akun terkunci
	err = db.QueryRow("SELECT Islogin, Expiredlock FROM Log WHERE Iduser = ? ORDER BY Idlog DESC LIMIT 1", userId).Scan(&isLogin, &expiredlock)
	if err == nil && expiredlock.Valid && time.Now().Before(expiredlock.Time) && isLogin == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Account locked. Try again after %v", expiredlock.Time)})
		return
	}

	// Cek apakah password benar
	err = db.QueryRow("SELECT Iduser FROM Tbluser WHERE Email = ? AND Password = ?", login.Email, login.Password).Scan(&userId)
	if err != nil {
		// Password salah, periksa jumlah percobaan yang gagal dari log sebelumnya
		var failedAttempts int
		err = db.QueryRow("SELECT COUNT(*) FROM Log WHERE Iduser = ? AND Islogin = 0 AND Expiredlock >= ?", userId, time.Now().Add(-5*time.Minute)).Scan(&failedAttempts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check login attempts"})
			return
		}

		// Jika gagal 3 kali, kunci akun selama 5 menit
		if failedAttempts >= 2 {
			expiredlock = sql.NullTime{Time: time.Now().Add(5 * time.Minute), Valid: true}
			_, err := db.Exec("INSERT INTO Log (Iduser, Islogin, Expiredlock) VALUES (?, 0, ?)", userId, expiredlock.Time)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update login attempts"})
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Too many failed attempts. Account locked for 5 minutes."})
		} else {
			// Update jumlah gagal login jika masih kurang dari 3 kali
			_, err := db.Exec("INSERT INTO Log (Iduser, Islogin, Expiredlock) VALUES (?, 0, NULL)", userId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update login attempts"})
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		}
		return
	}

	// Jika login berhasil, reset failed attempts dan expiredlock
	_, err = db.Exec("INSERT INTO Log (Iduser, Islogin, Expiredlock) VALUES (?, 1, NULL)", userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": userId})
}

func getUserData(c *gin.Context) {
	id := c.Param("id")

	var user struct {
		Namauser     string
		Email        string
		Alamatdetail string
		Provinsi     string
		Kabupaten    string
		Kodepos      string
	}

	err := db.QueryRow(`
		SELECT u.Namauser, u.Email, a.Alamatdetail, a.Provinsi, a.Kabupaten, a.Kodepos 
		FROM Tbluser u 
		JOIN Alamatuser a ON u.Iduser = a.Iduser 
		WHERE u.Iduser = ?`, id).Scan(&user.Namauser, &user.Email, &user.Alamatdetail, &user.Provinsi, &user.Kabupaten, &user.Kodepos)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user data"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func updateAddress(c *gin.Context) {
	id := c.Param("id")
	var address struct {
		Alamatdetail string `json:"alamatdetail"`
		Provinsi     string `json:"provinsi"`
		Kabupaten    string `json:"kabupaten"`
		Kodepos      string `json:"kodepos"`
	}

	if err := c.BindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		UPDATE Alamatuser SET Alamatdetail = ?, Provinsi = ?, Kabupaten = ?, Kodepos = ? 
		WHERE Iduser = ?`, address.Alamatdetail, address.Provinsi, address.Kabupaten, address.Kodepos, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address updated successfully"})
}

func main() {
	// Inisialisasi database
	initDB()

	// Inisialisasi router Gin
	router := gin.Default()

	// Route untuk registrasi user
	router.POST("/register", registerUser)

	// Route untuk login user
	router.POST("/login", loginUser)

	// Route untuk mengambil data user
	router.GET("/user/:id", getUserData)

	// Route untuk memperbarui data alamat user
	router.PUT("/user/:id/address", updateAddress)

	// Menjalankan server pada port 8080
	router.Run(":8080")
}
