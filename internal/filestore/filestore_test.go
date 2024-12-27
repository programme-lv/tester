package filestore_test

import (
	"os"
	"testing"

	"github.com/programme-lv/tester/internal/filestore"
	"github.com/stretchr/testify/require"
)

func TestFileStore(t *testing.T) {
	fileDir, err := os.MkdirTemp("", "filestore_test_files*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.RemoveAll(fileDir)

	downlDir, err := os.MkdirTemp("", "filestore_test_downl*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.RemoveAll(downlDir)

	fs := filestore.New(fileDir, downlDir)
	go fs.Start()

	err = fs.Schedule("572619f3013c7840cfd6113674ca0aefbdb573d4b334e3ee4e5be1642e27bd5a",
		"https://proglv-public.s3.eu-central-1.amazonaws.com/example-testfiles/572619f3013c7840cfd6113674ca0aefbdb573d4b334e3ee4e5be1642e27bd5a.zst")
	require.NoError(t, err)

	body, err := fs.Await("572619f3013c7840cfd6113674ca0aefbdb573d4b334e3ee4e5be1642e27bd5a")
	require.NoError(t, err)

	require.Equal(t, "315941512 -119267504\n", string(body))

	_, err = fs.Await("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.Error(t, err)

	// mismatch in integrity hash
	err = fs.Schedule("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"https://proglv-public.s3.eu-central-1.amazonaws.com/example-testfiles/16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce.zst")
	require.NoError(t, err)

	_, err = fs.Await("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	require.Error(t, err)

	err = fs.Schedule("16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce",
		"https://proglv-public.s3.eu-central-1.amazonaws.com/example-testfiles/16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce.zst")
	require.NoError(t, err)

	err = fs.Schedule("16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce",
		"https://proglv-public.s3.eu-central-1.amazonaws.com/example-testfiles/16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce.zst")
	require.NoError(t, err)

	body, err = fs.Await("16c8059bf85297cb959de90eed5b08f8742e48ac813f9228cbd6f8fb238478ce")
	require.NoError(t, err)

	require.Equal(t, "196674008\n", string(body))
}
