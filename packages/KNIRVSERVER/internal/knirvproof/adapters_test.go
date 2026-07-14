package knirvproof

import (
	"bytes"
	"context"
	"testing"
)

func TestFilesystemReplicatorCopiesAndVerifiesCiphertext(t *testing.T) {
	primary, err := NewFileStore(t.TempDir(), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	replicaOne, _ := NewFileStore(t.TempDir(), 1<<20)
	replicaTwo, _ := NewFileStore(t.TempDir(), 1<<20)
	ciphertext := []byte("encrypted-proof-object")
	cid := HashBytes(ciphertext)
	if _, _, err := primary.PutObject(context.Background(), cid, bytes.NewReader(ciphertext)); err != nil {
		t.Fatal(err)
	}
	objects := []ObjectRef{{CID: cid, Size: int64(len(ciphertext))}}
	storageRoot, _ := StorageRoot(objects)
	submission := ProofSubmission{Objects: objects, StorageRoot: storageRoot}
	replicator := FilesystemReplicator{
		PrimaryLocation: "primary", Replicas: []*FileStore{replicaOne, replicaTwo},
	}
	confirmations, err := replicator.Replicate(context.Background(), submission, primary)
	if err != nil {
		t.Fatal(err)
	}
	if len(confirmations) != 3 {
		t.Fatalf("confirmations = %d, want 3", len(confirmations))
	}
	for index, replica := range []*FileStore{replicaOne, replicaTwo} {
		exists, size, err := replica.HasObject(cid)
		if err != nil || !exists || size != int64(len(ciphertext)) {
			t.Fatalf("replica %d was not content-verified", index+1)
		}
	}
}
