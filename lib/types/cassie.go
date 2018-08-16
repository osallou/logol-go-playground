package logol

import (
    "log"
    cassie "org.irisa.genouest/cassiopee"
)

type CassieSearchOptions struct{
    Mode int
    MaxSubst int
    MaxIndel int
    Ambiguity bool
}

type Cassie struct {
    Indexer cassie.CassieIndexer
    Searcher cassie.CassieSearch
    PLen int
}

func NewCassieManager() Cassie {
    return Cassie{}
}

func (c Cassie) GetIndexer(sequence string) {
    c.Indexer = cassie.NewCassieIndexer(sequence)
    c.Indexer.SetMax_index_depth(1000)
    c.Indexer.SetMax_depth(10000)
    // c.Indexer.SetDo_reduction(true)
    c.Indexer.Index()
    c.Indexer.Graph()
    /*
    c.Searcher = cassie.NewCassieSearch(c.Indexer)
    c.Searcher.SetMode(0)
    c.Searcher.SetMax_subst(0)
    c.Searcher.SetMax_indel(0)
    c.Searcher.SetAmbiguity(false)
    log.Printf("Sequence %s indexed", sequence)
    */
}

func (c Cassie) GetSearcher(options CassieSearchOptions) {
    log.Printf("DEBUG load new searcher")
    c.Searcher = cassie.NewCassieSearch(c.Indexer)
    c.Searcher.SetMode(0)
    c.Searcher.SetMax_subst(0)
    c.Searcher.SetMax_indel(0)
    c.Searcher.SetAmbiguity(false)

}
