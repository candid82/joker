
(def syms '(set! ensure unchecked-remainder-int eduction aset aset-float deftype ->VecNode reduced? chunk-first sorted-map comparator chunk-cons unchecked-float proxy-call-with-super deliver unchecked-subtract file-seq char-array line-seq gen-class with-open conj! class biginteger alter unchecked-add compile .. pcalls memfn struct-map aset-double rsubseq with-loading-context proxy sorted? unquote-splicing tagged-literal byte-array unchecked-dec sorted-set extend await replicate bound-fn* hash-combine unchecked-inc-int ref-max-history vector-of Throwable->map set-error-handler! underive *print-level* add-watch *verbose-defrecords* aset-short *compile-files* float *math-context* construct-proxy agent-error agent-errors ifn? doto bound-fn defstruct print-simple bases aset-byte vswap! struct chunk-buffer chunk-next init-proxy longs unchecked-double await-for into-array future send-off ns-imports seque EMPTY-NODE vreset! char-escape-string with-local-vars thread-bound? chunk send-via hash-ordered-coll unchecked-byte subseq bytes unchecked-long *print-meta* to-array-2d set-error-mode! map-entry? ancestors set-agent-send-executor! error-handler *suppress-read* update-proxy hash-unordered-coll get-thread-bindings shorts ref-min-history create-struct completing int-array *unchecked-math* ref-set sorted-map-by *fn-loader* *use-context-classloader* await1 *print-length* future-cancel object-array accessor shutdown-agents print-ctor *default-data-reader-fn* areduce definline *compile-path* find-protocol-impl volatile? *print-dup* *compiler-options* unquote release-pending-sends re-matcher *agent* extends? supers byte unreduced floats disj! load-reader bean booleans locking error-mode decimal? set-validator! primitives-classnames *warn-on-reflection* alength restart-agent agent defprotocol send alter-var-root ints ->Eduction mix-collection-hash satisfies? reader-conditional bigdec to-array unchecked-subtract-int *allow-unresolved-vars* munge *source-path* extend-protocol unchecked-multiply-int aset-boolean chunk-rest isa? float-array *data-readers* future-cancelled? default-data-readers unchecked-multiply namespace-munge future-done? find-keyword ->VecSeq find-protocol-method *read-eval* extend-type aset-int pmap -cache-protocol-fn ensure-reduced reify *clojure-version* unchecked-int gen-interface unchecked-negate chars unchecked-short class? boolean-array ->ArrayChunk persistent! unchecked-dec-int extenders aset-char future? rationalize print-method remove-watch pop-thread-bindings proxy-name ref push-thread-bindings aget ref-history-count pvalues doubles assoc! get-validator definterface future-call long-array descendants resultset-seq add-classpath short char-name-string unchecked-add-int aclone reduced aset-long make-hierarchy dissoc! set-agent-send-off-executor! unchecked-inc clear-agent-errors reader-conditional? proxy-super unchecked-negate-int volatile! proxy-mappings enumeration-seq amap import short-array transient prefers compare-and-set! transduce unchecked-divide-int clojure-version iterator-seq unchecked-char derive chunk-append with-precision re-groups pop! commute get-proxy-class dosync method-sig sync sorted-set-by long make-array defrecord ->Vec tagged-literal? promise double-array print-dup parents record? -reset-methods io!))

(defn arg-decls
  [arglists]
  (if (= 1 (count arglists))
    (str (first arglists))
    (clojure.string/join " " (map #(str "(" % ")") arglists))))

(defn decls
  [sym]
  (let [v (ns-resolve 'clojure.core sym)
        m (meta v)
        arglists (:arglists m)]
    (when (and (seq arglists)
               (not (:macro m)))
      (str "(defn " sym " " (arg-decls arglists) ")"))))

(println (clojure.string/join "\n" (keep decls syms)))
