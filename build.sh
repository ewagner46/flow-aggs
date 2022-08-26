if [ -f flowaggs-server ]
then
  rm flowaggs-server
fi
cd cmd/flowaggs-server/
go build
cd ../../
cp cmd/flowaggs-server/flowaggs-server .
