package processor

import (
	"bytes"
	"sync"
)

// Used by trie structure to store the types
const (
	TString int = iota + 1
	TSlcomment
	TMlcomment
	TComplexity
)

// Language is a struct which contains the values for each language stored in languages.json
type Language struct {
	LineComment      []string   `json:"line_comment"`
	ComplexityChecks []string   `json:"complexitychecks"`
	Extensions       []string   `json:"extensions"`
	ExtensionFile    bool       `json:"extensionFile"`
	MultiLine        [][]string `json:"multi_line"`
	Quotes           [][]string `json:"quotes"`
	NestedMultiLine  bool       `json:"nestedmultiline"`
	Keywords         []string   `json:"keywords"`
}

// LanguageFeature is a struct which represents the conversion from Language into what is used for matching
type LanguageFeature struct {
	Complexity            *Trie
	MultiLineComments     *Trie
	SingleLineComments    *Trie
	Strings               *Trie
	Tokens                *Trie
	Nested                bool
	ComplexityCheckMask   byte
	SingleLineCommentMask byte
	MultiLineCommentMask  byte
	StringCheckMask       byte
	ProcessMask           byte
	Keywords              []string
}

// FileJobCallback is an interface that FileJobs can implement to get a per line callback with the line type
type FileJobCallback interface {
	// ProcessLine should return true to continue processing or false to stop further processing and return
	ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool
}

// FileJob is a struct used to hold all of the results of processing internally before sent to the formatter
type FileJob struct {
	Language           string
	PossibleLanguages  []string // Used to hold potentially more than one language which populates language when determined
	Filename           string
	Extension          string
	Location           string
	Content            []byte
	Bytes              int64
	Lines              int64
	Code               int64
	Comment            int64
	Blank              int64
	Complexity         int64
	WeightedComplexity float64
	Hash               []byte
	Callback           FileJobCallback
	Binary             bool
}

// LanguageSummary is used to hold summarised results for a single language
type LanguageSummary struct {
	Name               string
	Bytes              int64
	Lines              int64
	Code               int64
	Comment            int64
	Blank              int64
	Complexity         int64
	Count              int64
	WeightedComplexity float64
	Files              []*FileJob
}

// OpenClose is used to hold an open/close pair for matching such as multi line comments
type OpenClose struct {
	Open  []byte
	Close []byte
}

// CheckDuplicates is used to hold hashes if duplicate detection is enabled
type CheckDuplicates struct {
	hashes map[int64][][]byte
	mux    sync.Mutex
}

// Add concurrent safe add a key into the duplicates check
func (c *CheckDuplicates) Add(key int64, hash []byte) {
	c.mux.Lock()
	defer c.mux.Unlock()

	hashes, ok := c.hashes[key]
	if ok {
		c.hashes[key] = append(hashes, hash)
	} else {
		c.hashes[key] = [][]byte{hash}
	}
}

// Check concurrent safe check to see if the key exists already
func (c *CheckDuplicates) Check(key int64, hash []byte) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	hashes, ok := c.hashes[key]
	if ok {
		for _, h := range hashes {
			if bytes.Equal(h, hash) {
				return true
			}
		}
	}

	return false
}

// Trie is a structure used to store matches efficiently
type Trie struct {
	Type  int
	Close []byte
	Table [256]*Trie
}

// Insert inserts a string into the trie for matching
func (root *Trie) Insert(tokenType int, token []byte) {
	var node *Trie

	node = root
	for _, c := range token {
		if node.Table[c] == nil {
			node.Table[c] = &Trie{}
		}
		node = node.Table[c]
	}
	node.Type = tokenType
}

// InsertClose closes off a string in the trie
func (root *Trie) InsertClose(tokenType int, openToken, closeToken []byte) {
	var node *Trie

	node = root
	for _, c := range openToken {
		if node.Table[c] == nil {
			node.Table[c] = &Trie{}
		}
		node = node.Table[c]
	}
	node.Type = tokenType
	node.Close = closeToken
}

// Match checks the created trie structure for a match
func (root *Trie) Match(token []byte) (int, int, []byte) {
	var node *Trie
	var depth int
	var c byte

	node = root
	for depth, c = range token {
		if node.Table[c] == nil {
			return node.Type, depth, node.Close
		}
		node = node.Table[c]
	}
	return node.Type, depth, node.Close
}
