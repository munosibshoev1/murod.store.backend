package config

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var(
    Client *mongo.Client
    UserCollection *mongo.Collection
    CardCollection *mongo.Collection
    TransactionCollection *mongo.Collection
    CashierCollection *mongo.Collection
    StorekeeperCollection *mongo.Collection
    OperatorCollection *mongo.Collection
    ClientCollection *mongo.Collection
    SessionCollection *mongo.Collection
    ProductCollection *mongo.Collection
    ProductTemplateCollection *mongo.Collection
    CategoryCollection *mongo.Collection
    OrderCollection *mongo.Collection
    SupplierCollection *mongo.Collection
    SupplierOrderCollection *mongo.Collection
    PeshraftCollection *mongo.Collection
    ShopAPIKeyCollection *mongo.Collection
    SMSLogCollection *mongo.Collection
    TemporarySessionCollection *mongo.Collection
    OrderReturnCollection *mongo.Collection
    WriteOffCollection *mongo.Collection
    TransactionCollectionP *mongo.Collection
)
func ConnectDatabase() {
    client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGO_URI")))
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = client.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }

    err = client.Ping(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    Client = client
    UserCollection = Client.Database("murodstore").Collection("users")
    // CardCollection = Client.Database("store").Collection("cards")
    
    CashierCollection = Client.Database("murodstore").Collection("cashiers")
    StorekeeperCollection = Client.Database("murodstore").Collection("storekeepers")
    OperatorCollection = Client.Database("murodstore").Collection("operators")

    WriteOffCollection = Client.Database("murodstore").Collection("writeoffs")

    TransactionCollection = Client.Database("murodstore").Collection("transactions")
    ClientCollection = client.Database("murodstore").Collection("clients")
    SessionCollection = client.Database("murodstore").Collection("sessions")
    ProductCollection = client.Database("murodstore").Collection("product")
    ProductTemplateCollection= client.Database("murodstore").Collection("producttemplate")
    CategoryCollection = client.Database("murodstore").Collection("category")
    OrderCollection = client.Database("murodstore").Collection("orders")
    OrderReturnCollection = client.Database("murodstore").Collection("returnorders")
    SupplierCollection = client.Database("murodstore").Collection("suppliers")
    SupplierOrderCollection = client.Database("murodstore").Collection("supplierorders")
    ShopAPIKeyCollection = client.Database("murodstore").Collection("api")
    //api integration
    PeshraftCollection = Client.Database("murodpeshraft").Collection("cards")
    CardCollection = Client.Database("murodpeshraft").Collection("clients")
    TransactionCollectionP = Client.Database("murodpeshraft").Collection("transactions")

    SMSLogCollection = client.Database("murodstore").Collection("smsLog")
    TemporarySessionCollection = client.Database("murodstore").Collection("tempsessions")
    log.Println("Connected to MongoDB")
}
