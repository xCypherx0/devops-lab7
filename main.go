package main

import (
 "context"
 "encoding/json"
 "fmt"
 "log"
 "net"
 "net/http"
 "strconv"
 "strings"
 "sync"

 "github.com/go-kit/kit/endpoint"
 kitgrpc "github.com/go-kit/kit/transport/grpc"
 kithttp "github.com/go-kit/kit/transport/http"
 "google.golang.org/grpc"
 "google.golang.org/grpc/reflection"
 "google.golang.org/grpc/codes"
 "google.golang.org/grpc/status"
 "lab6/pb"
)

type Car struct { ID string `json:"id"`; Brand string `json:"brand"`; Model string `json:"model"`; Year int32 `json:"year"`; Price float64 `json:"price"`; Sold bool `json:"sold"` }
type Service interface { Create(context.Context, Car)(string,error); GetByID(context.Context,string)(Car,error); MarkSold(context.Context,string)error }
type repository struct { mu sync.RWMutex; cars map[string]Car; next int }
func (r *repository) Create(_ context.Context,c Car)(string,error) { r.mu.Lock(); defer r.mu.Unlock(); r.next++; c.ID=strconv.Itoa(r.next); r.cars[c.ID]=c; return c.ID,nil }
func (r *repository) GetByID(_ context.Context,id string)(Car,error) { r.mu.RLock(); defer r.mu.RUnlock(); c,ok:=r.cars[id]; if !ok { return Car{},status.Error(codes.NotFound,"car not found") }; return c,nil }
func (r *repository) MarkSold(_ context.Context,id string)error { r.mu.Lock(); defer r.mu.Unlock(); c,ok:=r.cars[id]; if !ok { return status.Error(codes.NotFound,"car not found") }; c.Sold=true; r.cars[id]=c; return nil }
type Endpoints struct { create, get, sell endpoint.Endpoint }
func makeEndpoints(s Service) Endpoints { return Endpoints{
 create: func(ctx context.Context,req interface{})(interface{},error) { id,e:=s.Create(ctx,req.(Car)); return map[string]string{"id":id},e },
 get: func(ctx context.Context,req interface{})(interface{},error) { return s.GetByID(ctx,req.(string)) },
 sell: func(ctx context.Context,req interface{})(interface{},error) { e:=s.MarkSold(ctx,req.(string)); return map[string]string{"message":"sold"},e },
} }
func main() {
 repo:=&repository{cars:make(map[string]Car)}; ep:=makeEndpoints(repo)
 enc:=func(_ context.Context,w http.ResponseWriter,v interface{})error { w.Header().Set("Content-Type","application/json"); return json.NewEncoder(w).Encode(v) }
 decCreate:=func(_ context.Context,r *http.Request)(interface{},error) { var c Car; err:=json.NewDecoder(r.Body).Decode(&c); return c,err }
 decID:=func(_ context.Context,r *http.Request)(interface{},error) { return strings.Split(strings.Trim(r.URL.Path,"/"),"/")[1],nil }
 create:=kithttp.NewServer(ep.create,decCreate,enc); get:=kithttp.NewServer(ep.get,decID,enc); sell:=kithttp.NewServer(ep.sell,decID,enc)
 mux:=http.NewServeMux(); mux.HandleFunc("/cars",func(w http.ResponseWriter,r *http.Request){ if r.Method!=http.MethodPost { w.WriteHeader(405); return }; create.ServeHTTP(w,r) })
 mux.HandleFunc("/cars/",func(w http.ResponseWriter,r *http.Request){ parts:=strings.Split(strings.Trim(r.URL.Path,"/"),"/"); if len(parts)==2 && r.Method==http.MethodGet { get.ServeHTTP(w,r); return }; if len(parts)==3 && parts[2]=="sold" && r.Method==http.MethodPatch { sell.ServeHTTP(w,r); return }; http.NotFound(w,r) })
 lis,err:=net.Listen("tcp",":9090"); if err!=nil { log.Fatal(err) }; gs:=grpc.NewServer(); pb.RegisterCarsServer(gs,newGRPC(ep)); reflection.Register(gs)
 go func(){ log.Println("gRPC :9090"); log.Fatal(gs.Serve(lis)) }()
 log.Println("HTTP :8080"); log.Fatal(http.ListenAndServe(":8080",mux))
}
type grpcServer struct { pb.UnimplementedCarsServer; create,get,sell kitgrpc.Handler }
func newGRPC(e Endpoints)*grpcServer {
 decCar:=func(_ context.Context,req interface{})(interface{},error){ c:=req.(*pb.Car); return Car{Brand:c.Brand,Model:c.Model,Year:c.Year,Price:c.Price,Sold:c.Sold},nil }
 decID:=func(_ context.Context,req interface{})(interface{},error){ return req.(*pb.ID).Id,nil }
 encID:=func(_ context.Context,res interface{})(interface{},error){ return &pb.ID{Id:res.(map[string]string)["id"]},nil }
 encCar:=func(_ context.Context,res interface{})(interface{},error){ c:=res.(Car); return &pb.Car{Id:c.ID,Brand:c.Brand,Model:c.Model,Year:c.Year,Price:c.Price,Sold:c.Sold},nil }
 encResult:=func(_ context.Context,res interface{})(interface{},error){ return &pb.Result{Message:fmt.Sprint(res.(map[string]string)["message"])},nil }
 return &grpcServer{create:kitgrpc.NewServer(e.create,decCar,encID),get:kitgrpc.NewServer(e.get,decID,encCar),sell:kitgrpc.NewServer(e.sell,decID,encResult)}
}
func (s *grpcServer) Create(ctx context.Context,r *pb.Car)(*pb.ID,error){ _,v,e:=s.create.ServeGRPC(ctx,r); if e!=nil{return nil,e}; return v.(*pb.ID),nil }
func (s *grpcServer) GetByID(ctx context.Context,r *pb.ID)(*pb.Car,error){ _,v,e:=s.get.ServeGRPC(ctx,r); if e!=nil{return nil,e}; return v.(*pb.Car),nil }
func (s *grpcServer) MarkSold(ctx context.Context,r *pb.ID)(*pb.Result,error){ _,v,e:=s.sell.ServeGRPC(ctx,r); if e!=nil{return nil,e}; return v.(*pb.Result),nil }
