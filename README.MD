    ├── config/           # 配置文件加载(后续可引入viper)
    ├── controller/       # 接口层：处理请求参数转换
    ├── middleware/       # 中间件：JWT验证，Cors，安全限流等
    ├── model/            # 数据库实体类
    ├── router/           # 路由层：定义API路径
    ├── service/          # 业务逻辑层：核心逻辑，操作数据库
    ├── utils/            # 工具类：加密、JWT生成等
    └── main.go           # 入口文件


