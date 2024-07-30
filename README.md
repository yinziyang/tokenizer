部署:
sudo cp libtokenizers.a /usr/local/lib

提示：
tokenizer.json下载地址：https://huggingface.co/google-bert/bert-base-multilingual-cased/raw/main/tokenizer.json

或者使用transformers生成:
```python
from transformers import  BertTokenizerFast

TOKENIZER_PATH = "./tokenizer"

tokenizer = BertTokenizerFast.from_pretrained('bert-base-multilingual-cased')
tokenizer.save_pretrained(TOKENIZER_PATH, legacy_format=False)
```

使用示例:
```go
package main

import (
    "github.com/yinziyang/tokenizer"
    "fmt"
)

const TOKENIZER_PATH = "tokenizer.json"
const MAX_LENGTH = 500
const NEED_PAD = false
const ADD_SPECIAL_TOKENS = true


func main() {
    tk, err := tokenizer.FromFile(TOKENIZER_PATH)
    if err != nil {
        panic(err)
    }
    defer tk.Close()

    fmt.Printf("%#v\n", tk.EncodeWithOptions(os.Args[1], MAX_LENGTH, NEED_PAD, ADD_SPECIAL_TOKENS, tokenizer.WithReturnAllAttributes()))
}
```
