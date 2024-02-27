注意，patch.go 主要是针对pb中enum类型增加显示的`MarshalJSON`接口，用于把enum类型按string序列化，用于适配http-arpc的enum处理行为。

工具参考[golang/protobuf](https://github.com/golang/protobuf)

