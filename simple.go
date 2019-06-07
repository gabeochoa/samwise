package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

// Database Models ....
// They come with ID, CreatedAt, UpdatedAt and DeletedAt for free
// Folder model for ORM
type Folder struct {
	gorm.Model
	Name string
}

// Record model for ORM
type Record struct {
	gorm.Model
	Folder Folder
	Key    string
	Data   []byte
}

func initDB() {
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Record{})
	db.AutoMigrate(&Folder{})

	// Apply extra rules

	// TODO: if we want cascase on delete folder
	//db.Model(&Profile{}).AddForeignKey("record_refer", "users(refer)", "CASCADE", "CASCADE")
}

// TODO: I feel like this could be a one liner...
func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func main() {
	// TODO: always migrate until we have data we want to save
	os.Remove("test.db")

	// Check to see if we have a db already
	// TODO: right now we have to delete to migrate (i think)
	if !fileExists("test.db") {
		fmt.Println("TestDB doesnt exist, creating and migrating")
		initDB()
	}
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Create
	db.Create(&Folder{
		Name: "test",
	})

	var foundFolder Folder
	db.First(&foundFolder, 0)
	foundFolderJ, _ := json.Marshal(foundFolder)
	fmt.Println("Printing the folder...")
	fmt.Println(string(foundFolderJ))

	db.Create(&Record{
		Folder: foundFolder,
		Key:    "example",
		Data:   []byte(`{"Name":"Bob","Food":"Pickle"}`),
	})

	// Read

	// TODO check searching
	// db.First(&product, "code = ?", "L1212") // find product with code l1212

	// find record with id 1
	var record Record
	db.First(&record, 1)
	recJ, _ := json.Marshal(record)
	fmt.Println(string(recJ))

	// Find the folder metadata associated with this
	var assocfolder Folder
	db.Model(&record).Related(&assocfolder)
	folderJ, _ := json.Marshal(assocfolder)
	fmt.Println(string(folderJ))

	// Update - update product's price to 2000
	// db.Model(&product).Update("Price", 2000)

	// Delete - delete product
	// db.Delete(&record)
	// db.Delete(&folder)
}
