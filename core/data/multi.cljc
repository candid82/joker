(defn ^:private check-valid-options-joke
  "Throws an exception if the given option map contains keys not listed
  as valid, else returns nil."
  [options & valid-keys]
  (when (seq (apply disj (apply hash-set (keys options)) valid-keys))
    (throw
      (ex-info ; was: (IllegalArgumentException.
        (apply str "Only these options are valid: "
          (first valid-keys)
          (map #(str ", " %) (rest valid-keys))) {}))))

;;multimethods
(defn- space-str
  "Quick and dirty version of joker.string/join with a single space as the separator, because joker.core shouldn't depend on joker.string"
  ([] "")
  ([a] (str " " a))
  ([a & more] (str " " a (apply space-str more))))

(defn- multifn-nomatch
  [name args]
  (throw (ex-info (str "No method in multimethod for dispatch value: (" name (apply space-str args) ")") {})))

(defn- new-multifn
  [name dispatch-fn default hierarchy]
;  (prn "after: name=" name " dispatch-fn=" dispatch-fn " default=" default " hierarchy=" hierarchy)
  (when hierarchy
    (throw (ex-info ":hierarchy not yet supported by joker.core/defmulti" {})))
  (let [mfatom (atom {})]
    (with-meta
      (fn [& args]
        ;;      (prn "mfatom=" @mfatom " dispatch-fn=" dispatch-fn " args=" args " default=" default)
        (let [method (get @mfatom
                          (apply dispatch-fn args)
                          (get @mfatom
                               default
                               (fn [& args] (multifn-nomatch name args))))]
          ;;        (prn "mfatom=" @mfatom " dispatch-fn=" dispatch-fn " args=" args " default=" default " method=" method)
          (apply method args)))
      {:name name :dispatch-fn dispatch-fn :default default :ns *ns* :method-table mfatom})))

(defn- multifn-swap-method-table-vals!
  [multifn ^Callable f & args]
  (let [mfm (meta multifn)
        mfatom (:method-table mfm)]
    (apply swap-vals! mfatom f args)))

(defn- multifn-reset
  [multifn]
  (let [mfm (meta multifn)
        mfatom (:method-table mfm)]
    (reset! mfatom {})))

(defn- multifn-remove-method
  [multifn dispatch-val]
  nil)

(defn- multifn-prefer-method
  [multifn dispatch-val-x dispatch-val-y]
  nil)

(defn- multifn-get-method-table
  [multifn]
  (let [mfm (meta multifn)
        mfatom (:method-table mfm)]
    @mfatom))

(defn- multifn-get-method
  [multifn dispatch-val]
  (let [mfm (meta multifn)
        mfatom (:method-table mfm)
        default (:default mfm)]
;    (prn "multifn-get-method multifn=" multifn " mfm=" mfm " mfatom=" mfatom " default=" default)
    (get @mfatom dispatch-val default)))

(defn- multifn-get-prefer-table
  [multifn]
  nil)

(defmacro defmulti-joke
  "Creates a new multimethod with the associated dispatch function.
  The docstring and attr-map are optional.

  Options are key-value pairs and may be one of:

  :default

  The default dispatch value, defaults to :default

  :hierarchy (UNSUPPORTED)

  The value used for hierarchical dispatch (e.g. ::square is-a ::shape)

  Hierarchies are type-like relationships that do not depend upon type
  inheritance. By default Clojure's multimethods dispatch off of a
  global hierarchy map.  However, a hierarchy relationship can be
  created with the derive function used to augment the root ancestor
  created with make-hierarchy.

  Multimethods expect the value of the hierarchy option to be supplied as
  a reference type e.g. a var (i.e. via the Var-quote dispatch macro #'
  or the var special form)."
  {:arglists '([name docstring? attr-map? dispatch-fn & options])
   :added "1.0"}
  [mm-name & options]
  (let [docstring   (if (string? (first options))
                      (first options)
                      nil)
        options     (if (string? (first options))
                      (next options)
                      options)
        m           (if (map? (first options))
                      (first options)
                      {})
        options     (if (map? (first options))
                      (next options)
                      options)
        dispatch-fn (first options)
        options     (next options)
        m           (if docstring
                      (assoc m :doc docstring)
                      m)
        m           (if (meta mm-name)
                      (conj (meta mm-name) m)
                      m)
        mm-name (with-meta mm-name m)]
    (when (= (count options) 1)
      (throw (ex-info "The syntax for defmulti has changed. Example: (defmulti name dispatch-fn :default dispatch-value)" {}))) ; was: (throw (Exception. ...))
    (let [options   (apply hash-map options)
          default   (get options :default :default)
          hierarchy (get options :hierarchy nil)]
;      (prn "before: mm-name=" mm-name " dispatch-fn=" dispatch-fn " default=" default " hierarchy=" hierarchy)
      (check-valid-options-joke options :default :hierarchy)
      `(let [v# (def ~mm-name)]
         (when-not (and (bound? v#) ; was: (and (.hasRoot v#) (instance? clojure.lang.MultiFn (deref v#)))
                        (fn? (deref v#))
                        (:method-table (meta (deref v#))))
           (let [fndef# (apply new-multifn (list (var ~mm-name) ~dispatch-fn ~default ~hierarchy))]
;             (prn "during: mm-name=" ~(name mm-name) " dispatch-fn=" ~dispatch-fn " default=" ~default " hierarchy=" ~hierarchy " fndef=" fndef#)
                 (def ~mm-name fndef#)))))))

(defmacro defmethod-joke
  "Creates and installs a new method of multimethod associated with dispatch-value. "
  {:added "1.0"}
  [multifn dispatch-val & fn-tail]
  `(multifn-swap-method-table-vals! ~multifn assoc ~dispatch-val (fn ~@fn-tail)))

(defn remove-all-methods-joke
  "Removes all of the methods of multimethod."
  {:added "1.2"
   :static true}
  [^clojure.lang.MultiFn multifn]
  (multifn-reset multifn)
  multifn)

(defn remove-method-joke
  "Removes the method of multimethod associated with dispatch-value."
  {:added "1.0"
   :static true}
 [^clojure.lang.MultiFn multifn dispatch-val]
 (multifn-remove-method multifn dispatch-val))

(defn prefer-method-joke
  "Causes the multimethod to prefer matches of dispatch-val-x over dispatch-val-y
   when there is a conflict"
  {:added "1.0"
   :static true}
  [^clojure.lang.MultiFn multifn dispatch-val-x dispatch-val-y]
  (multifn-prefer-method multifn dispatch-val-x dispatch-val-y))

(defn methods-joke
  "Given a multimethod, returns a map of dispatch values -> dispatch fns"
  {:added "1.0"
   :static true}
  [multifn] ; was: [^clojure.lang.MultiFn multifn]
  (multifn-get-method-table multifn))

(defn get-method-joke
  "Given a multimethod and a dispatch value, returns the dispatch fn
  that would apply to that value, or nil if none apply and no default"
  {:added "1.0"
   :static true}
  [multifn dispatch-val] ; was: [^clojure.lang.MultiFn multifn...]
  (multifn-get-method multifn dispatch-val))

(defn prefers-joke
  "Given a multimethod, returns a map of preferred value -> set of other values"
  {:added "1.0"
   :static true}
  [multifn] ; was: [^clojure.lang.MultiFn multifn]
  (multifn-get-prefer-table multifn))