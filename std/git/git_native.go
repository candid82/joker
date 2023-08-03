package git

import (
	"unsafe"

	. "github.com/candid82/joker/core"
	git "github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type (
	GitRepo struct {
		repo *git.Repository
		hash uint32
	}
)

var gitRepoType *Type

func MakeGitRepo(repo *git.Repository) GitRepo {
	res := GitRepo{repo, 0}
	res.hash = HashPtr(uintptr(unsafe.Pointer(repo)))
	return res
}

func (repo GitRepo) ToString(_escape bool) string {
	return "#object[GitRepo]"
}

func (repo GitRepo) Equals(other interface{}) bool {
	if other, ok := other.(GitRepo); ok {
		return repo.repo == other.repo
	}
	return false
}

func (repo GitRepo) GetInfo() *ObjectInfo {
	return nil
}

func (repo GitRepo) GetType() *Type {
	return gitRepoType
}

func (repo GitRepo) Hash() uint32 {
	return repo.hash
}

func (repo GitRepo) WithInfo(_info *ObjectInfo) Object {
	return repo
}

func EnsureArgIsGitRepo(args []Object, index int) GitRepo {
	obj := args[index]
	if c, yes := obj.(GitRepo); yes {
		return c
	}
	panic(FailArg(obj, "GitRepo", index))
}

func ExtractGitRepo(args []Object, index int) *git.Repository {
	return EnsureArgIsGitRepo(args, index).repo
}

func open(path string) *git.Repository {
	repo, err := git.PlainOpen(path)
	PanicOnErr(err)
	return repo
}

func addUser(m *ArrayMap, section string, name string, email string) {
	user := EmptyArrayMap()
	user.Add(MakeKeyword("name"), MakeString(name))
	user.Add(MakeKeyword("email"), MakeString(email))
	m.Add(MakeKeyword(section), user)
}

func makeRemote(remote *gitConfig.RemoteConfig) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeString(remote.Name))
	res.Add(MakeKeyword("urls"), MakeStringVector(remote.URLs))
	res.Add(MakeKeyword("mirror?"), MakeBoolean(remote.Mirror))
	return res
}

func makeSubmodule(submodule *gitConfig.Submodule) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeString(submodule.Name))
	res.Add(MakeKeyword("path"), MakeString(submodule.Path))
	res.Add(MakeKeyword("url"), MakeString(submodule.URL))
	res.Add(MakeKeyword("branch"), MakeString(submodule.Branch))
	return res
}

func makeBranch(branch *gitConfig.Branch) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeString(branch.Name))
	res.Add(MakeKeyword("remote"), MakeString(branch.Remote))
	res.Add(MakeKeyword("merge"), MakeString(string(branch.Merge)))
	res.Add(MakeKeyword("rebase"), MakeString(branch.Rebase))
	res.Add(MakeKeyword("description"), MakeString(branch.Description))
	return res
}

func makeUrl(url *gitConfig.URL) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeString(url.Name))
	res.Add(MakeKeyword("instead-of"), MakeString(url.InsteadOf))
	return res
}

func config(repo *git.Repository) Map {
	cfg, err := repo.Config()
	PanicOnErr(err)
	res := EmptyArrayMap()
	res.Add(MakeKeyword("bare?"), MakeBoolean(cfg.Core.IsBare))
	res.Add(MakeKeyword("worktree"), MakeString(cfg.Core.Worktree))
	res.Add(MakeKeyword("comment-char"), MakeString(cfg.Core.CommentChar))
	res.Add(MakeKeyword("repository-format-version"), MakeString(string(cfg.Core.RepositoryFormatVersion)))
	res.Add(MakeKeyword("default-branch"), MakeString(cfg.Init.DefaultBranch))
	addUser(res, "user", cfg.User.Name, cfg.User.Email)
	addUser(res, "author", cfg.Author.Name, cfg.Author.Email)
	addUser(res, "committer", cfg.Committer.Name, cfg.Committer.Email)
	remotes := EmptyArrayMap()
	for name, remote := range cfg.Remotes {
		remotes.Add(MakeString(name), makeRemote(remote))
	}
	res.Add(MakeKeyword("remotes"), remotes)
	submodules := EmptyArrayMap()
	for name, submodule := range cfg.Submodules {
		submodules.Add(MakeString(name), makeSubmodule(submodule))
	}
	res.Add(MakeKeyword("submodules"), submodules)
	branches := EmptyArrayMap()
	for name, branch := range cfg.Branches {
		branches.Add(MakeString(name), makeBranch(branch))
	}
	res.Add(MakeKeyword("branches"), branches)
	urls := EmptyArrayMap()
	for name, url := range cfg.URLs {
		branches.Add(MakeString(name), makeUrl(url))
	}
	res.Add(MakeKeyword("urls"), urls)
	return res
}

func makeRef(r *plumbing.Reference) Map {
	res := EmptyArrayMap()
	refType := "invalid"
	if r.Type() == plumbing.HashReference {
		refType = "hash"
	} else if r.Type() == plumbing.SymbolicReference {
		refType = "symbolic"
	}
	res.Add(MakeKeyword("type"), MakeKeyword(refType))
	res.Add(MakeKeyword("name"), MakeString(string(r.Name())))
	res.Add(MakeKeyword("target"), MakeString(string(r.Target())))
	res.Add(MakeKeyword("hash"), MakeString(r.Hash().String()))
	return res
}

func ref(repo *git.Repository, name string, resolved bool) Map {
	r, err := repo.Reference(plumbing.ReferenceName(name), resolved)
	PanicOnErr(err)
	return makeRef(r)
}

func head(repo *git.Repository) Map {
	r, err := repo.Head()
	PanicOnErr(err)
	return makeRef(r)
}

func makeSignature(s object.Signature) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("name"), MakeKeyword(s.Name))
	res.Add(MakeKeyword("email"), MakeKeyword(s.Email))
	res.Add(MakeKeyword("when"), MakeTime(s.When))
	return res
}

func makeCommit(cmt *object.Commit) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("hash"), MakeString(cmt.Hash.String()))
	res.Add(MakeKeyword("author"), makeSignature(cmt.Author))
	res.Add(MakeKeyword("committer"), makeSignature(cmt.Committer))
	res.Add(MakeKeyword("pgp-siganture"), MakeString(cmt.PGPSignature))
	res.Add(MakeKeyword("message"), MakeString(cmt.Message))
	res.Add(MakeKeyword("tree-hash"), MakeString(cmt.TreeHash.String()))
	parentHashes := EmptyVector()
	for _, v := range cmt.ParentHashes {
		parentHashes = parentHashes.Conjoin(MakeString(v.String()))
	}
	res.Add(MakeKeyword("parent-hashes"), parentHashes)
	return res
}

func log(repo *git.Repository, opts Map) Vec {
	var logOpts git.LogOptions
	if ok, v := opts.Get(MakeKeyword("from")); ok {
		logOpts.From = plumbing.NewHash(EnsureObjectIsString(v, "").S)
	}
	if ok, v := opts.Get(MakeKeyword("order")); ok {
		if v.Equals(MakeKeyword("default")) {
			logOpts.Order = git.LogOrderDefault
		} else if v.Equals(MakeKeyword("dfs")) {
			logOpts.Order = git.LogOrderDFS
		} else if v.Equals(MakeKeyword("dfs-post")) {
			logOpts.Order = git.LogOrderDFSPost
		} else if v.Equals(MakeKeyword("bsf")) {
			logOpts.Order = git.LogOrderBSF
		} else if v.Equals(MakeKeyword("committer-time")) {
			logOpts.Order = git.LogOrderCommitterTime
		} else {
			panic(RT.NewError(":order must be one of: :default, :dfs, :dfs-post, :bsf, :committer-time"))
		}
	}
	if ok, v := opts.Get(MakeKeyword("path-filter")); ok {
		fn := EnsureObjectIsFn(v, "Invalid :path-filter option: %s")
		logOpts.PathFilter = func(s string) bool {
			return ToBool(fn.Call([]Object{MakeString(s)}))
		}
	}
	if ok, v := opts.Get(MakeKeyword("all")); ok {
		logOpts.All = ToBool(v)
	}
	if ok, v := opts.Get(MakeKeyword("since")); ok {
		t := EnsureObjectIsTime(v, "Invalid :since option: %s").T
		logOpts.Since = &t
	}
	if ok, v := opts.Get(MakeKeyword("until")); ok {
		t := EnsureObjectIsTime(v, "Invalid :until option: %s").T
		logOpts.Until = &t
	}
	it, err := repo.Log(&logOpts)
	PanicOnErr(err)
	res := EmptyArrayVector()
	err = it.ForEach(func(cmt *object.Commit) error {
		res.Append(makeCommit(cmt))
		return nil
	})
	PanicOnErr(err)
	return res
}

func resolveRevision(repo *git.Repository, rev string) string {
	hash, err := repo.ResolveRevision(plumbing.Revision(rev))
	PanicOnErr(err)
	return hash.String()
}

func findCommit(repo *git.Repository, hash string) Map {
	obj, err := repo.CommitObject(plumbing.NewHash(hash))
	PanicOnErr(err)
	return makeCommit(obj)
}

func findObject(repo *git.Repository, hash string) Map {
	obj, err := repo.Object(plumbing.AnyObject, plumbing.NewHash(hash))
	PanicOnErr(err)
	res := EmptyArrayMap()
	res.Add(MakeKeyword("id"), MakeString(obj.ID().String()))
	objType := MakeKeyword("invalid")
	switch obj.Type() {
	case 1:
		objType = MakeKeyword("commit")
	case 2:
		objType = MakeKeyword("tree")
	case 3:
		objType = MakeKeyword("blob")
	case 4:
		objType = MakeKeyword("tag")
	case 6:
		objType = MakeKeyword("ofs-delta")
	case 7:
		objType = MakeKeyword("ref-delta")
	case 8:
		objType = MakeKeyword("any")
	}
	res.Add(MakeKeyword("type"), objType)
	return res
}

func status(repo *git.Repository) Map {
	workTree, err := repo.Worktree()
	PanicOnErr(err)
	st, err := workTree.Status()
	PanicOnErr(err)
	res := EmptyArrayMap()
	for filename, filestatus := range st {
		fileStatusMap := EmptyArrayMap()
		fileStatusMap.Add(MakeKeyword("staging"), MakeChar(rune(filestatus.Staging)))
		fileStatusMap.Add(MakeKeyword("worktree"), MakeChar(rune(filestatus.Worktree)))
		fileStatusMap.Add(MakeKeyword("extra"), MakeString(filestatus.Extra))
		res.Add(MakeString(filename), fileStatusMap)
	}
	return res
}

func init() {
	gitRepoType = RegType("GitRepo", (*GitRepo)(nil), "Wraps git.Repository type")
}
