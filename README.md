# TestCaseGolang

Nama Kandidat : Fathin Akhimullah \n
Posisi : Backend Developer	
Penggunaan Service : 
**GOLANG**
Route untuk registrasi user
 - router.POST("/register", registerUser) // Menambahkan data baru

  - Route untuk login user
	- router.POST("/login", loginUser) // Mengecek data yang diinput apakah sesuai dengan data yang sudah diregister atau belum.

  - Route untuk mengambil data user
	- router.GET("/user/:id", getUserData) // Menampilkan data yang sudah ditambahkan ke database

   - Route untuk memperbarui data alamat user
	- router.PUT("/user/:id/address", updateAddress) // Merubah data alamat yang sudah ada di database
