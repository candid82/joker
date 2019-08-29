(def ^:private stream (js/require "stream"))
(def ^:private ReadableStream (.-ReadableStream stream))
(ReadableStream. #js {:read (fn [_] ())})
