package routes

import (
	"net/http"

	"project-empat-golang/controllers"
	"project-empat-golang/middlewares"
)

func Routes(mux *http.ServeMux) {

	// Login & Register routes
	mux.HandleFunc("POST /api/register", controllers.Register)
	mux.HandleFunc("POST /api/login", controllers.Login)

	// Auth routes - user yang sudah login
	mux.HandleFunc("GET /api/me", middlewares.AuthMiddleware(controllers.GetMe))
	mux.HandleFunc("PUT /api/me", middlewares.AuthMiddleware(controllers.UpdateProfile))
	mux.HandleFunc("POST /api/logout", middlewares.AuthMiddleware(controllers.Logout))

	// Public routes - semua user bisa akses
	mux.HandleFunc("GET /api/categories", controllers.GetAllCategories)
	mux.HandleFunc("GET /api/categories/{id}", controllers.GetCategoryByID)

	// Admin only routes - hanya admin yang bisa create, update, delete
	mux.HandleFunc("POST /api/categories", middlewares.RoleMiddleware(controllers.CreateCategory, "admin"))
	mux.HandleFunc("PUT /api/categories/{id}", middlewares.RoleMiddleware(controllers.UpdateCategory, "admin"))
	mux.HandleFunc("DELETE /api/categories/{id}", middlewares.RoleMiddleware(controllers.DeleteCategory, "admin"))

	//Public routes - semua user bisa akses
	mux.HandleFunc("GET /api/products", controllers.GetAllProducts)
	mux.HandleFunc("GET /api/products/{id}", controllers.GetProductByID)

	// Admin only routes - hanya admin yang bisa create, update, delete
	mux.HandleFunc("POST /api/products", middlewares.RoleMiddleware(controllers.CreateProduct, "admin"))
	mux.HandleFunc("PUT /api/products/{id}", middlewares.RoleMiddleware(controllers.UpdateProduct, "admin"))
	mux.HandleFunc("DELETE /api/products/{id}", middlewares.RoleMiddleware(controllers.DeleteProduct, "admin"))

	// Cart routes - user yang sudah login
	mux.HandleFunc("GET /api/cart", middlewares.AuthMiddleware(controllers.GetCart))
	mux.HandleFunc("POST /api/cart/items", middlewares.AuthMiddleware(controllers.AddToCart))
	mux.HandleFunc("PUT /api/cart/items/{id}", middlewares.AuthMiddleware(controllers.UpdateCartItem))
	mux.HandleFunc("DELETE /api/cart/items/{id}", middlewares.AuthMiddleware(controllers.DeleteCartItem))
	mux.HandleFunc("DELETE /api/cart", middlewares.AuthMiddleware(controllers.ClearCart))

	// Checkout & Order routes - user yang sudah login
	mux.HandleFunc("POST /api/checkout", middlewares.AuthMiddleware(controllers.Checkout))
	mux.HandleFunc("GET /api/orders", middlewares.AuthMiddleware(controllers.GetMyOrders))
	mux.HandleFunc("GET /api/orders/{id}", middlewares.AuthMiddleware(controllers.GetOrderByID))
}
