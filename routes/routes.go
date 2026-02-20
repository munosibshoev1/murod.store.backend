package routes

import (
	"backend/controllers"
	"backend/handlers"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

func InitializeRoutes(router *gin.Engine) {
	router.POST("/login", controllers.Login)
	router.POST("/forgot-password", controllers.RequestPasswordReset)
	router.POST("/verify-code", controllers.VerifyCode)
	router.POST("/reset-password", controllers.ResetPassword)
	router.POST("/registration", controllers.RegisterClient)
	router.Static("/uploads", "./uploads")
	router.GET("/getorder/:orderID", controllers.GetOrderByID1)
	router.GET("/getorderbyclient/:clientID", controllers.GetOrderByID1)
	router.GET("/getlocation", controllers.GetStorekeeperLocations)
	router.GET("/getdeliverylocation", controllers.GetDeliveryLocations)
	router.GET("/order/:token", handlers.GetOrderByToken)   
	
	client := router.Group("/client")
	client.Use(middleware.AuthMiddleware("client"))
	{
		client.GET("/my-card/:id", handlers.GetTransactionsByCardClient)
		client.GET("/my-data/:id", handlers.GetClientInfo)
		client.PUT("/my-data/:id", handlers.UpdateClientInfo)
		client.GET("/category/select", controllers.GetAllCategories)
		client.GET("/category", controllers.GetAllCategories1)
		client.GET("/category/:id", handlers.GetCategoryDetails)
		client.GET("/getdeliverylocation", controllers.GetDeliveryLocations)
		client.GET("/getlimit/:cardnumber", handlers.GetCardLimit)
		client.GET("/products", controllers.GetAllProducts)
		client.GET("/getlocation", controllers.GetStorekeeperLocations)
		client.POST("/order", handlers.CreateCustomerOrder)
		client.GET("/orderbyid/:id", controllers.GetOrderByID)
		client.GET("/order/:clientid", controllers.GetOrdersByCustomerID)
	}

	admin := router.Group("/admin")
	admin.Use(middleware.AuthMiddleware("admin"))
	{
		admin.GET("/clients", controllers.ListClients)
		admin.POST("/clients", controllers.AddClient)
		admin.GET("/clientl", controllers.ListRetailClients)
		admin.POST("/writeoffs", controllers.WriteOffProductsNEW)
		admin.PUT("/writeoffs/:id", controllers.ConfirmWriteOff)
		admin.GET("/writeoffs", controllers.GetWriteOffDocuments)
		admin.GET("/writeoffs/:id", controllers.GetWriteOffDocumentByID)

		admin.GET("/operators", controllers.ListOperators)
		admin.POST("/operators", controllers.AddOperator)

		admin.GET("/storekeepers", controllers.ListStorekeepers)
		admin.POST("/storekeepers", controllers.AddStorekeeper)
		admin.PUT("/storekeepers/:id", controllers.UpdateStorekeeper)
		admin.GET("/storekeepers/:id", controllers.GetStorekeeper)

		admin.GET("/cashiers", controllers.ListCashiers)
		admin.POST("/cashiers", controllers.AddCashier)
		admin.GET("/cards", controllers.ListCards)
		admin.POST("/cards/generate", controllers.GenerateCards)
		admin.GET("/clients/:id", controllers.GetClient)
		admin.GET("/cashiers/:id", controllers.GetCashier)
		admin.DELETE("/clients/:id", controllers.DeleteClient)
		admin.DELETE("/cashiers/:id", controllers.DeleteCashier)
		admin.PUT("/clients/:id", controllers.UpdateClient)
		admin.PUT("/cashiers/:id", controllers.UpdateCashier)
		admin.GET("/transactions", controllers.GetAllTransactions)
		admin.GET("/cardReport", controllers.GetCardReport)
		admin.GET("/unusedcards", controllers.GetUnusedCards)

		//store product templates routes
		admin.POST("/addproducttemp", controllers.AddProductTemplate)
		admin.PUT("/editproducttemp/:id", controllers.EditProductTemplate)
		admin.DELETE("/productstemp/:id", controllers.DeleteProductTemplate)
		admin.GET("/producttemp/:id", controllers.GetProductTemplate)
		admin.GET("/productstemp/:id", controllers.GetProductTemplate)
		admin.GET("/productstemp", controllers.GetAllProductsTemplate)
		
		admin.PUT("/editproduct/:id", controllers.EditProduct)
		admin.DELETE("/products/:id", controllers.DeleteProduct)
		admin.GET("/products/:id", controllers.GetProduct)
		admin.GET("/productrefund/:id", controllers.GetProduct)
        admin.PUT("/productrefund/:id", controllers.ReturnCustomerOrderPartial)
		
		//store category routes
		admin.POST("/addcategory", controllers.CreateCategory)
		admin.PUT("/editcategory/:id", controllers.EditCategory)
		admin.DELETE("/categorys/:id", controllers.DeleteCategory)
		admin.GET("/categorys/:id", controllers.GetCategory)
		admin.GET("/getcategory/:id", controllers.GetCategory)
		admin.GET("/categorys", controllers.GetAllCategories)
		admin.GET("/category/select", controllers.GetAllCategories)
		admin.GET("/products", controllers.GetAllProductsAdmin)
		admin.GET("/products/select", controllers.GetProductsWithSelectedFields)
		//supplier routes
		admin.POST("/addsupplier", controllers.CreateSupplier)
		admin.PUT("/editsupplier/:id", controllers.EditSupplier)
		admin.DELETE("/suppliers/:id", controllers.DeleteSupplier)
		admin.GET("/getsupplier/:id", controllers.GetSupplier)
		admin.GET("/suppliers/:id", controllers.GetSupplier)
		admin.GET("/suppliers", controllers.GetAllSuppliersNEW)
		admin.GET("/supplier/select", controllers.GetSuppliersForSelect)
		admin.POST("/addsupplierorder", controllers.AddSupplierOrder)

		admin.GET("/supplierorders/:id", controllers.GetSupplierOrder) // Полуить заказ по ID
		admin.GET("/supplierorders", controllers.GetSupplierOrders)    // Получить все заказы (с фильтром по SupplierID)
		admin.PUT("/supplierorders/:id", controllers.EditSupplierOrder)
		
		admin.PUT("/storekeepers/:id", controllers.EditSupplierOrderS)
		admin.PUT("/setprice/:id", controllers.EditSupplierOrderSelling1)

		admin.GET("/clientorder/:id", controllers.GetOrderByID)
		admin.DELETE("/clientorder/:id", controllers.ReturnCustomerOrder)  // Получение заказа по ID
		admin.GET("/clientorder", controllers.GetAllOrdersNEW)                 // Получение всех заказов
		admin.GET("/customer/:clientid", controllers.GetOrdersByCustomerID) // Получение заказов клиента
		admin.GET("/clientreturnorders", controllers.GetAllReturnOrders)  
		admin.GET("/clientreturnorders/:id", controllers.GetReturnOrderByID) 
		admin.PUT("/clientorder/:id", handlers.AdminUpdateCustomerOrder) 
		// admin.GET("/productreport/:barcode", controllers.GetProductSalesReport) 
		admin.GET("/productreport/:barcode", controllers.GetProductSalesReportNEW) 
		admin.GET("/dashboard", controllers.Dashboard) 

	}

	storekeeper := router.Group("/storekeeper")
	storekeeper.Use(middleware.AuthMiddleware("storekeeper"))
	{

		storekeeper.GET("/orders/:id", controllers.GetSupplierOrderStorekeeper) // Получить заказ по ID
		storekeeper.GET("/supplierorders", controllers.GetSupplierOrders)
		storekeeper.GET("/orders", controllers.GetSupplierOrders1) // Получить все заказы (с фильтром по SupplierID)
		storekeeper.PUT("/orders/:id", controllers.EditSupplierOrderS)
		storekeeper.GET("/clientorder", controllers.GetAllOrders)
		storekeeper.GET("/clientorder/:id", controllers.GetOrderByID)
		storekeeper.PUT("/clientorder/:id", controllers.ConfirmCustomerOrder)

		storekeeper.GET("/clientreturnorders", controllers.GetAllReturnOrders)  
		storekeeper.GET("/clientreturnorders/:id", controllers.GetReturnOrderByID)
		storekeeper.PUT("/clientreturnorders/:id", controllers.ConfirmReturnToStock) 
		storekeeper.GET("/products", controllers.GetProductsWithSelectedFields)
		storekeeper.GET("/products/select", controllers.GetProductsWithSelectedFields)

		storekeeper.POST("/writeoffs", controllers.WriteOffProductsNEW)
		storekeeper.PUT("/writeoffs/:id", controllers.UpdateWriteOffDraft)
		
		storekeeper.GET("/writeoffs", controllers.GetWriteOffDocuments)
		storekeeper.GET("/writeoffs/:id", controllers.GetWriteOffDocumentByID)
		// storekeeper.GET("/clientreturnorder/:id", controllers.GetReturnOrderByID) 
		
	}
	operator := router.Group("/operator/")
	operator.Use(middleware.AuthMiddleware("operator"))
	{
		operator.GET("/suppliers", controllers.GetAllSuppliersNEW)	
		operator.GET("/supplier/select", controllers.GetSuppliersForSelect)

		operator.GET("/productstemp", controllers.GetAllProductsTemplate)
		operator.POST("/addproducttemp", controllers.AddProductTemplate)
		operator.GET("/producttemp/:id", controllers.GetProductTemplate)
		operator.GET("/productstemp/:id", controllers.GetProductTemplate)
		operator.POST("/addsupplierorder", controllers.AddSupplierOrderNEW)
		operator.GET("/category/select", controllers.GetAllCategories)
		operator.GET("/supplierorders/:id", controllers.GetSupplierOrderNEW) // Полуить заказ по ID
		operator.GET("/supplierorders", controllers.GetSupplierOrders)    
	}
	// Добавляем роут для интернет-магазина
	api := router.Group("/api")
	api.Use(middleware.DynamicAPIKeyMiddleware()) // Динамическая проверка API-ключа
	{
		api.POST("/login", controllers.LoginCashierByCard)
	}
}
