package server

import (
	"authentication-service/api"
	"authentication-service/app"
	"authentication-service/grpcservice"
	acProtobuf "authentication-service/proto/v1/pb/account"
	authProtobuf "authentication-service/proto/v1/pb/authentication"
	"authentication-service/util"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "serves the sidekiq api",
		RunE:  run,
	}
}

func run(cmd *cobra.Command, args []string) error {

	//People service grpc client
	accountServiceClient, err := grpc.Dial("localhost:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer accountServiceClient.Close()

	app, err := app.New()
	if err != nil {
		return err
	}
	defer app.Close()

	//Set account service client
	app.Repos.AccountServiceClient = acProtobuf.NewAccountServiceClient(accountServiceClient)

	api, err := api.New(app)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	grpcctx, grpccancel := context.WithCancel(context.Background())

	go func() {
		defer util.RecoverGoroutinePanic(nil)
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		<-ch
		logrus.Info("signal caught. shutting down...")
		cancel()
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		defer cancel()
		serveAPI(ctx, api)
	}()

	wg.Add(2)
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		defer wg.Done()
		defer grpccancel()
		serveGrpc(grpcctx, app)
	}()

	wg.Wait()
	return nil
}

func serveAPI(ctx context.Context, api *api.API) {
	cors := handlers.CORS(
		handlers.AllowCredentials(),
		handlers.AllowedOrigins([]string{"http://localhost:3000", "*", "https://api-staging.sidekiq.com"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Cookie", "X-Requested-With", "ETag", "Profile", "Origin", "BoardID", "rs-sidkiq-auth-token", "Sec-Ch-Ua-Platform", "Sec-Ch-Ua-Mobile", "Sec-Ch-Ua", "Sec-Fetch-Dest", "Sec-Fetch-Mode", "Sec-Fetch-Site", "User-Agent"}),
	)

	router := mux.NewRouter()
	router.Use(cors)
	api.Init(router.PathPrefix("/api").Subrouter().StrictSlash(true))

	fs := http.FileServer(http.Dir("./public"))
	router.PathPrefix("/").Handler(fs)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", api.Config.Port),
		Handler:      router,
		ReadTimeout:  api.Config.ReadTimeout * time.Second,
		WriteTimeout: api.Config.WriteTimeout * time.Second,
	}

	done := make(chan struct{})
	go func() {
		defer util.RecoverGoroutinePanic(nil)
		<-ctx.Done()
		if err := s.Shutdown(context.Background()); err != nil {
			logrus.Error(err)
		}
		close(done)
	}()

	logrus.Infof("serving api at http://127.0.0.1:%d", api.Config.Port)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		logrus.Fatal(err)
	}
	<-done
}

func serveGrpc(ctx context.Context, app *app.App) {

	//Auth GRPC run on 8084 port
	listener, err := net.Listen("tcp", ":8084")
	if err != nil {
		logrus.Fatal(err)
	}

	//Create grpc server and register methods.
	s := grpc.NewServer()
	authProtobuf.RegisterAuthServiceServer(s, &grpcservice.AuthServer{
		App: app,
	})

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		s.GracefulStop()
		close(done)
	}()
	logrus.Infof("authentication-service GRPC STARTED at http://127.0.0.1:%d", 8084)
	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	<-done
}
