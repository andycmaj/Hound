package searcher

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/etsy/hound/config"
	"github.com/etsy/hound/index"
	"github.com/etsy/hound/vcs"
)

type Searcher struct {
	idx  *index.Index
	lck  sync.RWMutex
	Repo *config.Repo

	// The channel is used to request updates from the API and
	// to signal that it is ok for searchers to begin polling.
	// It has a buffer size of 1 to allow at most one pending
	// update at a time.
	updateCh chan time.Time
}

type limiter chan bool

/**
 * Holds a set of IndexRefs that were found in the dbpath at startup,
 * these indexes can be 'claimed' and re-used by newly created searchers.
 */
type foundRefs struct {
	refs    []*index.IndexRef
	claimed map[*index.IndexRef]bool
}

func makeLimiter(n int) limiter {
	return limiter(make(chan bool, n))
}

func (l limiter) Acquire() {
	l <- true
}

func (l limiter) Release() {
	<-l
}

/**
 * Find an Index ref for the repo url and rev, returns nil if no such
 * ref exists.
 */
func (r *foundRefs) find(url, rev string) *index.IndexRef {
	for _, ref := range r.refs {
		if ref.Url == url && ref.Rev == rev {
			return ref
		}
	}
	return nil
}

/**
 * Claim a ref for reuse. This ensures they ref will not be garbage
 * collected at the end of startup.
 */
func (r *foundRefs) claim(ref *index.IndexRef) {
	r.claimed[ref] = true
}

/**
 * Delete the directorires associated with all IndexRefs that were
 * found in the dbpath but were not claimed during startup.
 */
func (r *foundRefs) removeUnclaimed() error {
	for _, ref := range r.refs {
		if r.claimed[ref] {
			continue
		}

		if err := ref.Remove(); err != nil {
			return err
		}
	}
	return nil
}

// Perform atomic swap of index in the searcher so that the new
// index is made "live".
func (s *Searcher) swapIndexes(idx *index.Index) error {
	s.lck.Lock()
	defer s.lck.Unlock()

	oldIdx := s.idx
	s.idx = idx

	return oldIdx.Destroy()
}

// Perform a basic search on the current index using the supplied pattern
// and the options.
//
// TODO(knorton): pat should really just be a part of SearchOptions
func (s *Searcher) Search(pat string, opt *index.SearchOptions) (*index.SearchResponse, error) {
	s.lck.RLock()
	defer s.lck.RUnlock()
	return s.idx.Search(pat, opt)
}

// Get the excluded files as a JSON string. This is only used for returning
// the data directly to clients (thus JSON).
func (s *Searcher) GetExcludedFiles() string {
	path := filepath.Join(s.idx.GetDir(), "excluded_files.json")
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Couldn't read excluded_files.json %v\n", err)
	}
	return string(dat)
}

// Triggers an immediate poll of the repository.
func (s *Searcher) Update() bool {
	if !s.Repo.PushUpdatesEnabled() {
		return false
	}

	// schedule an update if one is not already scheduled
	select {
	case s.updateCh <- time.Now():
	default:
		// don't wait to enqueue another update
	}

	return true
}

// Wait for either the delay period to expire or an update request to
// arrive. Note that an empty delay will result in an infinite timeout.
func (s *Searcher) waitForUpdate(delay time.Duration) {
	var tch <-chan time.Time
	if delay.Nanoseconds() > 0 {
		tch = time.After(delay)
	}

	// wait for either the timeout or the update channel signal
	select {
	case <-s.updateCh:
	case <-tch:
	}
}

// Signal the searcher that it is ok to begin polling the repository.
func (s *Searcher) begin() {
	s.updateCh <- time.Now()
}

// Generate a new index directory in the dbpath. The names are based
// on pseudo-randomness with a time-based seed.
func nextIndexDir(dbpath string) string {
	r := uint64(rand.Uint32())<<32 | uint64(rand.Uint32())
	return filepath.Join(dbpath, fmt.Sprintf("idx-%08x", r))
}

// Read the refs associated with each of the index dirs
// in the given dbpath.
func findExistingRefs(dbpath string) (*foundRefs, error) {
	dirs, err := filepath.Glob(filepath.Join(dbpath, "idx-*"))
	if err != nil {
		return nil, err
	}

	var refs []*index.IndexRef
	for _, dir := range dirs {
		r, _ := index.Read(dir)
		refs = append(refs, r)
	}

	return &foundRefs{
		refs:    refs,
		claimed: map[*index.IndexRef]bool{},
	}, nil
}

// Open an index at the given path. If the idxDir is already present, it will
// simply open and use that index. If, however, the idxDir does not exist a new
// one will be built.
func buildAndOpenIndex(
	opt *index.IndexOptions,
	dbpath,
	vcsDir,
	idxDir,
	url,
	rev string) (*index.Index, error) {
	if _, err := os.Stat(idxDir); err != nil {
		r, err := index.Build(opt, idxDir, vcsDir, url, rev)
		if err != nil {
			return nil, err
		}

		return r.Open()
	}

	return index.Open(idxDir)
}

// Simply prints out statistics about the heap. When hound rebuilds a new
// index it will expand the heap with a decent amount of garbage. This is
// helpful to ensure the heap growth looks sane.
func reportOnMemory() {
	var ms runtime.MemStats

	// Print out interesting heap info.
	runtime.ReadMemStats(&ms)
	fmt.Printf("HeapInUse = %0.2f\n", float64(ms.HeapInuse)/1e6)
	fmt.Printf("HeapIdle  = %0.2f\n", float64(ms.HeapIdle)/1e6)
}

// Utility function for producing a hex encoded sha1 hash for a string.
func hashFor(name string) string {
	h := sha1.New()
	h.Write([]byte(name))
	return hex.EncodeToString(h.Sum(nil))
}

// Create a normalized name for the vcs directory of this repo.
func vcsDirFor(repo *config.Repo) string {
	return fmt.Sprintf("vcs-%s", hashFor(repo.Url))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Make a searcher for each repo in the Config. This function kind of has a notion
// of partial errors. First, if the error returned is non-nil then a fatal error has
// occurred and no other return values are valid. If an error occurs that is specific
// to a particular searcher, that searcher will not be present in the searcher map and
// will have an error entry in the error map.
//
// NOTE: The keys in the searcher map will be normalized to lower case, but not such
// transformation will be done on the error map to make it easier to match those errors
// back to the original repo name.
func MakeAll(cfg *config.Config) (map[string]*Searcher, map[string]error, error) {
	errs := map[string]error{}
	searchers := map[string]*Searcher{}

	refs, err := findExistingRefs(cfg.DbPath)
	if err != nil {
		return nil, nil, err
	}

	lim := makeLimiter(cfg.MaxConcurrentIndexers)

	for name, repo := range cfg.Repos {
		s, err := newSearcher(cfg.DbPath, name, repo, refs, lim)
		if err != nil {
			log.Print(err)
			errs[name] = err
			continue
		}

		searchers[strings.ToLower(name)] = s
	}

	if err := refs.removeUnclaimed(); err != nil {
		return nil, nil, err
	}

	// after all the repos are in good shape, we start their polling
	for _, s := range searchers {
		s.begin()
	}

	return searchers, errs, nil
}

// Creates a new Searcher that is available for searches as soon as this returns.
// This will pull or clone the target repo and start watching the repo for changes.
func New(dbpath, name string, repo *config.Repo) (*Searcher, error) {
	s, err := newSearcher(dbpath, name, repo, &foundRefs{}, makeLimiter(1))
	if err != nil {
		return nil, err
	}

	s.begin()

	return s, nil
}

// Update the vcs and reindex the given repo.
func updateAndReindex(
	s *Searcher,
	dbpath,
	vcsDir,
	name,
	rev string,
	wd *vcs.WorkDir,
	opt *index.IndexOptions,
	lim limiter) (string, bool) {

	// acquire a token from the rate limiter
	lim.Acquire()
	defer lim.Release()

	repo := s.Repo
	newRev, err := wd.PullOrClone(vcsDir, repo.Url)

	if err != nil {
		log.Printf("vcs pull error (%s - %s): %s", name, repo.Url, err)
		return rev, false
	}

	if newRev == rev {
		return rev, false
	}

	log.Printf("Rebuilding %s for %s", name, newRev)
	idx, err := buildAndOpenIndex(
		opt,
		dbpath,
		vcsDir,
		nextIndexDir(dbpath),
		repo.Url,
		newRev)
	if err != nil {
		log.Printf("failed index build (%s): %s", name, err)
		return rev, false
	}

	if err := s.swapIndexes(idx); err != nil {
		log.Printf("failed index swap (%s): %s", name, err)
		if err := idx.Destroy(); err != nil {
			log.Printf("failed to destroy index (%s): %s\n", name, err)
		}
		return rev, false
	}

	return newRev, true
}

// Creates a new Searcher that is capable of re-claiming an existing index directory
// from a set of existing manifests.
func newSearcher(
	dbpath, name string,
	repo *config.Repo,
	refs *foundRefs,
	lim limiter) (*Searcher, error) {

	vcsDir := filepath.Join(dbpath, vcsDirFor(repo))

	log.Printf("Searcher started for %s", name)

	wd, err := vcs.New(repo.Vcs, repo.VcsConfig())
	if err != nil {
		return nil, err
	}

	opt := &index.IndexOptions{
		ExcludeDotFiles: repo.ExcludeDotFiles,
		SpecialFiles:    wd.SpecialFiles(),
	}

	rev, err := wd.PullOrClone(vcsDir, repo.Url)
	if err != nil {
		return nil, err
	}

	var idxDir string
	ref := refs.find(repo.Url, rev)
	if ref == nil {
		idxDir = nextIndexDir(dbpath)
	} else {
		idxDir = ref.Dir()
		refs.claim(ref)
	}

	idx, err := buildAndOpenIndex(
		opt,
		dbpath,
		vcsDir,
		idxDir,
		repo.Url,
		rev)
	if err != nil {
		return nil, err
	}

	s := &Searcher{
		idx:      idx,
		updateCh: make(chan time.Time, 1),
		Repo:     repo,
	}

	go func() {

		// each searcher's poller is held until begin is called.
		<-s.updateCh

		// if all forms of updating are turned off, we're done here.
		if !repo.PollUpdatesEnabled() && !repo.PushUpdatesEnabled() {
			return
		}

		var delay time.Duration
		if repo.PollUpdatesEnabled() {
			delay = time.Duration(repo.MsBetweenPolls) * time.Millisecond
		}

		for {

			// Wait for a signal to proceed
			s.waitForUpdate(delay)

			// attempt to update and reindex this searcher
			newRev, ok := updateAndReindex(s, dbpath, vcsDir, name, rev, wd, opt, lim)
			if !ok {
				continue
			}

			rev = newRev

			// This is just a good time to GC since we know there will be a
			// whole set of dead posting lists on the heap. Ensuring these
			// go away quickly helps to prevent the heap from expanding
			// uncessarily.
			runtime.GC()

			reportOnMemory()
		}
	}()

	return s, nil
}
