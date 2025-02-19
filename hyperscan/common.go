package hyperscan

import (
	"fmt"
	"regexp"
)

// Database is an immutable database that can be used by the Hyperscan scanning API.
type Database interface {
	// Provides information about a database.
	Info() (DbInfo, error)

	// Provides the size of the given database in bytes.
	Size() (int, error)

	// Free a compiled pattern database.
	Close() error

	// Serialize a pattern database to a stream of bytes.
	Marshal() ([]byte, error)

	// Reconstruct a pattern database from a stream of bytes at a given memory location.
	Unmarshal([]byte) error
}

// BlockDatabase scan the target data that is a discrete,
// contiguous block which can be scanned in one call and does not require state to be retained.
type BlockDatabase interface {
	Database
	BlockScanner
	BlockMatcher
}

// StreamDatabase scan the target data to be scanned is a continuous stream,
// not all of which is available at once;
// blocks of data are scanned in sequence and matches may span multiple blocks in a stream.
type StreamDatabase interface {
	Database
	StreamScanner
	StreamMatcher
	StreamCompressor

	StreamSize() (int, error)
}

// VectoredDatabase scan the target data that consists of a list of non-contiguous blocks
// that are available all at once.
type VectoredDatabase interface {
	Database
	VectoredScanner
	VectoredMatcher
}

const infoMatches = 4

var regexInfo = regexp.MustCompile(`^Version: (\d+\.\d+\.\d+) Features: ([\w\s]+)? Mode: (\w+)$`)

// DbInfo identify the version and platform information for the supplied database.
type DbInfo string // nolint: stylecheck

func (i DbInfo) String() string { return string(i) }

// Version is the version for the supplied database.
func (i DbInfo) Version() (string, error) {
	matched := regexInfo.FindStringSubmatch(string(i))

	if len(matched) != infoMatches {
		return "", fmt.Errorf("database info, %w", ErrInvalid)
	}

	return matched[1], nil
}

// Mode is the scanning mode for the supplied database.
func (i DbInfo) Mode() (ModeFlag, error) {
	matched := regexInfo.FindStringSubmatch(string(i))

	if len(matched) != infoMatches {
		return 0, fmt.Errorf("database info, %w", ErrInvalid)
	}

	return ParseModeFlag(matched[3])
}

// Version identify this release version. The return version is a string
// containing the version number of this release build and the date of the build.
func Version() string { return hsVersion() }

// ValidPlatform test the current system architecture.
func ValidPlatform() error { return hsValidPlatform() }

type database interface {
	Db() hsDatabase
}

type baseDatabase struct {
	db hsDatabase
}

func newBaseDatabase(db hsDatabase) *baseDatabase {
	return &baseDatabase{db}
}

// UnmarshalDatabase reconstruct a pattern database from a stream of bytes.
func UnmarshalDatabase(data []byte) (Database, error) {
	db, err := hsDeserializeDatabase(data)
	if err != nil {
		return nil, err
	}

	return &baseDatabase{db}, nil
}

// UnmarshalBlockDatabase reconstruct a block database from a stream of bytes.
func UnmarshalBlockDatabase(data []byte) (BlockDatabase, error) {
	db, err := hsDeserializeDatabase(data)
	if err != nil {
		return nil, err
	}

	return newBlockDatabase(db), nil
}

// UnmarshalStreamDatabase reconstruct a stream database from a stream of bytes.
func UnmarshalStreamDatabase(data []byte) (StreamDatabase, error) {
	db, err := hsDeserializeDatabase(data)
	if err != nil {
		return nil, err
	}

	return newStreamDatabase(db), nil
}

// UnmarshalVectoredDatabase reconstruct a vectored database from a stream of bytes.
func UnmarshalVectoredDatabase(data []byte) (VectoredDatabase, error) {
	db, err := hsDeserializeDatabase(data)
	if err != nil {
		return nil, err
	}

	return newVectoredDatabase(db), nil
}

// SerializedDatabaseSize reports the size that would be required by a database if it were deserialized.
func SerializedDatabaseSize(data []byte) (int, error) { return hsSerializedDatabaseSize(data) }

// SerializedDatabaseInfo provides information about a serialized database.
func SerializedDatabaseInfo(data []byte) (DbInfo, error) {
	i, err := hsSerializedDatabaseInfo(data)

	return DbInfo(i), err
}

func (d *baseDatabase) Db() hsDatabase { return d.db } // nolint: stylecheck

func (d *baseDatabase) Size() (int, error) { return hsDatabaseSize(d.db) }

func (d *baseDatabase) Info() (DbInfo, error) {
	i, err := hsDatabaseInfo(d.db)

	return DbInfo(i), err
}

func (d *baseDatabase) Close() error { return hsFreeDatabase(d.db) }

func (d *baseDatabase) Marshal() ([]byte, error) { return hsSerializeDatabase(d.db) }

func (d *baseDatabase) Unmarshal(data []byte) error { return hsDeserializeDatabaseAt(data, d.db) }

type blockDatabase struct {
	*blockMatcher
}

func newBlockDatabase(db hsDatabase) *blockDatabase {
	return &blockDatabase{newBlockMatcher(newBlockScanner(newBaseDatabase(db)))}
}

type streamDatabase struct {
	*streamMatcher
}

func newStreamDatabase(db hsDatabase) *streamDatabase {
	return &streamDatabase{newStreamMatcher(newStreamScanner(newBaseDatabase(db)))}
}

func (db *streamDatabase) StreamSize() (int, error) { return hsStreamSize(db.db) }

type vectoredDatabase struct {
	*vectoredMatcher
}

func newVectoredDatabase(db hsDatabase) *vectoredDatabase {
	return &vectoredDatabase{newVectoredMatcher(newVectoredScanner(newBaseDatabase(db)))}
}
