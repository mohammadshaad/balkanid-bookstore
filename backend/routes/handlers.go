package routes

import (
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mohammadshaad/golang-book-store-backend/database"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-playground/validator/v10"

	"github.com/golang-jwt/jwt/v4"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func LoginHandler(c *fiber.Ctx) error {
	var userData struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Validate user input
	if err := validate.Struct(userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid input data",
			"errors":  err.(validator.ValidationErrors),
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().Where("email = ?", userData.Email).First(&user).Error; err != nil {
		// Handle database errors (e.g., no user with the given email)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Compare the given password with the password in the database
	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(userData.Password)); err != nil {
		// Handle password incorrect error
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Incorrect password",
		})
	}

	// Create a JWT token
	token, err := CreateToken(user.ID)
	if err != nil {
		// Handle token creation error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot log in",
		})
	}

	// Return the token
	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
	})

}

func RegisterHandler(c *fiber.Ctx) error {
	var userData struct {
		FirstName string            `json:"firstname" validate:"required"`
		LastName  string            `json:"lastname" validate:"required"`
		Email     string            `json:"email" validate:"required,email"`
		Password  string            `json:"password" validate:"required"`
		Role      database.UserRole `json:"role" validate:"required"`
	}

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Validate user input
	if err := validate.Struct(userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Invalid input data",
			"errors":  err.(validator.ValidationErrors),
		})
	}

	// Check if the user already exists (email must be unique)
	var user database.User
	if err := database.GetDB().Where("email = ?", userData.Email).First(&user).Error; err == nil {
		// User already exists, don't register again
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User already exists",
		})
	}

	// Generate a random numeric user ID
	rand.Seed(time.Now().UnixNano())
	userID := uint(rand.Intn(10000)) // Change the range as needed

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"message": "Cannot hash password",
		})
	}

	// Create a new user with the generated ID
	newUser := database.User{
		UserID:    userID,
		FirstName: userData.FirstName,
		LastName:  userData.LastName,
		Email:     userData.Email,
		Password:  hashedPassword,
		Role:      userData.Role,
	}

	// Save the user to the database
	if err := database.GetDB().Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User registration failed",
		})
	}

	// Retrieve the auto-generated ID from the database
	autoGeneratedID := newUser.ID

	// Create a JWT token
	token, err := CreateToken(autoGeneratedID)
	if err != nil {
		// Handle token creation error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot log in",
		})
	}

	// Return the token
	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
	})

}

func DeactivateAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Deactivate the user
	if err := database.GetDB().Model(&user).Update("active", false).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot deactivate user",
		})
	}

	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deactivated successfully",
	})
}

func ActivateAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Activate the user
	if err := database.GetDB().Model(&user).Update("active", true).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot activate user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User activated successfully",
	})
}

func DeleteAccountHandler(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Delete the user's account from the database
	if err := database.GetDB().Delete(&user).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot delete user account",
		})
	}

	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User account deleted successfully",
	})
}

// Get users name
func GetUserNameHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"name":    user.FirstName,
	})
}

// User Home Page - Send the name of the logged in user in the response body
func UserHomePageHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"name":    user.FirstName,
	})
}

func LogoutHandler(c *fiber.Ctx) error {
	// Set the token's expiration time to now thereby invalidating it
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})

	// Return a success response
	return c.JSON(fiber.Map{
		"success": true,
		"message": "User logged out successfully",
	})
}

func Profile(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

func UpdateProfile(c *fiber.Ctx) error {
	// Get the "id" URL parameter and convert it to a uint
	idStr := c.Params("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, uint(id)).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var userData database.User

	if err := c.BodyParser(&userData); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Cannot parse JSON",
		})
	}

	// Update the user's first name if it's provided in the request
	if userData.FirstName != "" {
		user.FirstName = userData.FirstName
	}

	// Update the user's last name if it's provided in the request
	if userData.LastName != "" {
		user.LastName = userData.LastName
	}

	// Update the user's email if it's provided in the request
	if userData.Email != "" {
		user.Email = userData.Email
	}

	// Update the user's password if it's provided in the request
	if len(userData.Password) > 0 {
		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), 10)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"success": false,
				"message": "Cannot hash password",
			})
		}
		user.Password = hashedPassword
	}

	if err := database.GetDB().Save(&user).Error; err != nil {
		// Handle database errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Cannot update user's profile",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User profile updated successfully",
	})
}

// Create a new book
func CreateBookHandler(c *fiber.Ctx) error {
	var newBook database.Book
	if err := c.BodyParser(&newBook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Generate a random numeric book ID
	rand.Seed(time.Now().UnixNano())
	bookID := uint(rand.Intn(10000)) // Change the range as needed

	// Set the generated ID for the new book
	newBook.ID = bookID

	// Save the new book to the database
	if err := database.GetDB().Create(&newBook).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create book",
		})
	}
	return c.JSON(newBook)
}

// Get a list of all books or a single book by ID
func GetAllBooksHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	if id == "" {
		// No ID parameter, fetch all books
		var books []database.Book
		if err := database.GetDB().Find(&books).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch books",
			})
		}
		// Return books as a JSON object with a 'books' property
		return c.JSON(fiber.Map{
			"books": books,
		})
	}

	// ID parameter is present, fetch a single book by ID
	var book database.Book
	if err := database.GetDB().First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	return c.JSON(book)
}

// Get a single book by ID
func GetBookByIDHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var book database.Book
	if err := database.GetDB().First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}
	return c.JSON(book)
}

// Update a book by ID
func UpdateBookHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var updatedBook database.Book
	if err := c.BodyParser(&updatedBook); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Find the book in the database
	var book database.Book
	if err := database.GetDB().First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Update the book's information
	book.Title = updatedBook.Title
	book.Author = updatedBook.Author
	book.ISBN = updatedBook.ISBN
	book.Genre = updatedBook.Genre
	book.Price = updatedBook.Price
	book.Quantity = updatedBook.Quantity
	book.Description = updatedBook.Description
	book.Image = updatedBook.Image
	book.Path = updatedBook.Path

	// Save the updated book to the database
	if err := database.GetDB().Save(&book).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update book",
		})
	}

	return c.JSON(book)
}

// Delete a book by ID
func DeleteBookHandler(c *fiber.Ctx) error {
	id := c.Params("id")

	// Find the book in the database
	var book database.Book
	if err := database.GetDB().First(&book, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Delete the book from the database
	if err := database.GetDB().Delete(&book).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete book",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Book deleted successfully",
	})
}

// Get all users
func GetAllUsersHandler(c *fiber.Ctx) error {
	var users []database.User
	if err := database.GetDB().Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}
	return c.JSON(users)
}

// Get a single user by ID
func GetUserByIDHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	var user database.User
	if err := database.GetDB().First(&user, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	return c.JSON(user)
}

// Create JWT token
func CreateToken(userID uint) (string, error) {
	// Define the payload
	payload := jwt.MapClaims{}
	payload["user_id"] = userID
	payload["exp"] = time.Now().Add(time.Hour * 24).Unix() // 24 hours

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	// Generate the encoded token
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// Create a new cart item and add it to the user's cart
func AddToCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID and quantity from the request body
	var cartItem struct {
		BookID   uint `json:"book_id" validate:"required"`
		Quantity uint `json:"quantity" validate:"required"`
	}

	if err := c.BodyParser(&cartItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Validate the input
	if err := validate.Struct(cartItem); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid input data",
			"errors": err.(validator.ValidationErrors),
		})
	}

	// Check if the book is already in the user's cart
	var existingCartItem database.CartItem
	if err := database.GetDB().Where("user_id = ? AND book_id = ?", userID, cartItem.BookID).First(&existingCartItem).Error; err == nil {
		// Book is already in the cart, update the quantity
		existingCartItem.Quantity += cartItem.Quantity

		// Retrieve the book price
		var book database.Book
		if err := database.GetDB().First(&book, cartItem.BookID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch book details",
			})
		}

		// Calculate the subtotal and assign it to the existing cart item
		existingCartItem.Subtotal = float64(existingCartItem.Quantity) * book.Price

		if err := database.GetDB().Save(&existingCartItem).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update cart",
			})
		}
		return c.JSON(existingCartItem)
	}

	// Book is not in the cart, create a new cart item
	newCartItem := database.CartItem{
		UserID:   userID,
		BookID:   cartItem.BookID,
		Quantity: cartItem.Quantity,
	}

	// Retrieve the book price
	var book database.Book
	if err := database.GetDB().First(&book, cartItem.BookID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch book details",
		})
	}

	// Calculate the subtotal and assign it to the new cart item
	newCartItem.Subtotal = float64(newCartItem.Quantity) * book.Price

	if err := database.GetDB().Create(&newCartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add to cart",
		})
	}

	return c.JSON(newCartItem)
}

// Get the user's cart items
func GetCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Find all cart items for the user
	var cartItems []database.CartItem
	if err := database.GetDB().Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Remove an item from the user's cart
func RemoveFromCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find the cart item to remove
	var cartItem database.CartItem
	if err := database.GetDB().Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Delete the cart item
	if err := database.GetDB().Delete(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove item from cart",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Item removed from cart",
	})
}

// Update the quantity of a cart item
func UpdateCartItemQuantityHandler(c *fiber.Ctx) error {
	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Parse the new quantity from the request body
	var update struct {
		Quantity uint `json:"quantity" validate:"required"`
	}

	if err := c.BodyParser(&update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Validate the input
	if err := validate.Struct(update); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Invalid input data",
			"errors": err.(validator.ValidationErrors),
		})
	}

	// Find the cart item to update
	var cartItem database.CartItem
	if err := database.GetDB().Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Update the quantity
	cartItem.Quantity = update.Quantity
	if err := database.GetDB().Save(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update cart item quantity",
		})
	}

	return c.JSON(cartItem)
}

// Add a review for a book
func AddReviewHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookIDStr := c.Params("book_id")

	// Convert the book ID to a uint
	bookID, err := strconv.ParseUint(bookIDStr, 10, 32)
	if err != nil {
		// Handle invalid ID format
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID format",
		})
	}

	// Convert the book ID to a uint
	bookIDUint := uint(bookID)

	// Parse the user ID from the JWT token
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["user_id"].(float64))

	// Check if the user has already reviewed the book
	var existingReview database.Review
	if err := database.GetDB().Where("user_id = ? AND book_id = ?", userID, bookIDUint).First(&existingReview).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "You have already reviewed this book",
		})
	}

	// Check if the book exists
	var book database.Book
	if err := database.GetDB().First(&book, bookIDUint).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Check if the user exists
	var user database.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Parse the review data from the request body
	var review database.Review
	if err := c.BodyParser(&review); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input data",
		})
	}

	// Set the book ID and user ID
	review.BookID = bookIDUint
	review.UserID = userID

	// Save the review to the database
	if err := database.GetDB().Create(&review).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add review",
		})
	}

	// Fetch the review again from the database to get the created_at value
	if err := database.GetDB().Where("id = ?", review.ID).First(&review).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch review",
		})
	}

	return c.JSON(review)
}

// Get reviews for a book with user names
func GetBookReviewsHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find all reviews for the book and include user information
	var reviews []struct {
		database.Review
		FirstName string `json:"first_name"`
		CreatedAt string `json:"created_at"`
	}
	if err := database.GetDB().Table("reviews").
		Select("reviews.*, users.first_name, reviews.created_at").
		Joins("LEFT JOIN users ON users.id = reviews.user_id").
		Where("reviews.book_id = ?", bookID).
		Scan(&reviews).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch reviews",
		})
	}

	if len(reviews) == 0 {
		return c.JSON(fiber.Map{
			"message": "No reviews for this book",
		})
	}

	// Return the reviews with user first names and CreatedAt
	return c.JSON(reviews)
}

func DownloadBookHandler(c *fiber.Ctx) error {
	// Parse the book ID from the URL parameter
	bookID := c.Params("id")

	// Find the book in the database by ID
	var book database.Book
	if err := database.GetDB().First(&book, bookID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Book not found",
		})
	}

	// Get the file path
	filePath := book.Path

	// send the file path as a response
	return c.JSON(fiber.Map{
		"file_path": filePath,
	})
}

// Cart section for admin to see all the users cart items
func GetAllCartItemsHandler(c *fiber.Ctx) error {
	var cartItems []database.CartItem
	if err := database.GetDB().Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Get a user's cart items
func GetUserCartHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("user_id")

	// Find all cart items for the user
	var cartItems []database.CartItem
	if err := database.GetDB().Where("user_id = ?", userID).Find(&cartItems).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch cart items",
		})
	}

	if len(cartItems) == 0 {
		return c.JSON(fiber.Map{
			"message": "Cart is empty",
		})
	}

	// Return the cart items
	return c.JSON(cartItems)
}

// Remove an item from the user's cart
func DeleteCartItemHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("user_id")

	// Parse the book ID from the URL parameter
	bookID := c.Params("book_id")

	// Find the cart item to remove
	var cartItem database.CartItem
	if err := database.GetDB().Where("user_id = ? AND book_id = ?", userID, bookID).First(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Cart item not found",
		})
	}

	// Delete the cart item
	if err := database.GetDB().Delete(&cartItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove item from cart",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Item removed from cart",
	})
}

// Get the role of the user from the database
func GetUserRoleHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("id")

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}
	return c.JSON(fiber.Map{
		"role": user.Role,
	})
}

// Delete user handler
func DeleteUserHandler(c *fiber.Ctx) error {
	// Parse the user ID from the URL parameter
	userID := c.Params("id")

	// Find the user in the database
	var user database.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		// Handle database errors (e.g., no user with the given ID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Delete the user from the database
	if err := database.GetDB().Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}
