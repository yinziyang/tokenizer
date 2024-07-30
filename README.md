### 部署:
```sh
sudo cp libtokenizers.a /usr/local/lib
```

### 提示：
> tokenizer.json下载地址：https://huggingface.co/google-bert/bert-base-multilingual-cased/raw/main/tokenizer.json

> 或者使用transformers生成:
```python
from transformers import  BertTokenizerFast

TOKENIZER_PATH = "./tokenizer"

tokenizer = BertTokenizerFast.from_pretrained('bert-base-multilingual-cased')
tokenizer.save_pretrained(TOKENIZER_PATH, legacy_format=False)
```

### 使用示例
```go
package main

import (
    "fmt"
    "os"

    "github.com/yinziyang/tokenizer"
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

    fmt.Printf("%+v\n", tk.EncodeWithOptions(os.Args[1], MAX_LENGTH, NEED_PAD, ADD_SPECIAL_TOKENS, tokenizer.WithReturnAllAttributes()))
}
```

```sh
go run tokenizer.go "测试一下"

{IDs:[101 4988 7333 2072 2079 102] TypeIDs:[0 0 0 0 0 0] SpecialTokensMask:[1 0 0 0 0 1] AttentionMask:[1 1 1 1 1 1] Tokens:[[CLS] 测 试 一 下 [SEP]]}
```
