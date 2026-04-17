    ├── config/           # 配置文件加载(后续可引入viper)
    ├── controller/       # 接口层：处理请求参数转换
    ├── middleware/       # 中间件：JWT验证，Cors，安全限流等
    ├── model/            # 数据库实体类
    ├── router/           # 路由层：定义API路径
    ├── service/          # 业务逻辑层：核心逻辑，操作数据库
    ├── utils/            # 工具类：加密、JWT生成等
    └── main.go           # 入口文件

# docker exec -it kingbase bash

ksql -Usystem -d test -p54321

# 用新路径重新启动容器

docker run -dit \
--name postgis-server \
--restart=always \
--privileged=true \
-p 5432:5432 \
-e POSTGRES_PASSWORD=your_strong_password \
-e PGDATA=/var/lib/postgresql/data/pg_root \
-v /mnt/vos-vguapnz1/postgis/data:/var/lib/postgresql/data/pg_root:z \
postgis/postgis:18-3.6

# 非国产

docker run -dit \
--name postgis-server \
--restart=always \
-p 5432:5432 \
-e POSTGRES_PASSWORD=your_strong_password \
-v /data/postgis_data:/var/lib/postgresql \
postgis/postgis:18-3.6

# 2. 进入容器测试连接

docker exec -it postgis-server psql -U postgres

systemctl restart FS_Backend.service
systemctl status FS_Backend.service

### 进入容器

sudo docker exec -it postgis-server bash
后,使用shp2pgsql,raster2pgsql

#### shp2pgsql -s 4326 -I my_file.shp public.table_a | psql -d field_survey_db -U postgres

    shp2pgsql -s 4326 -I "geo_location/2021年县界.shp" public.geo_location | psql -d field_survey_db -U postgres
    shp2pgsql -s 4326 -I "watershed/全国三级流域空间分布基础数据.shp" public.watershed | psql -d field_survey_db -U postgres
    shp2pgsql -s 4326 -I "eco_zone/全国三级生态区.shp" public.eco_zone | psql -d field_survey_db -U postgres
    shp2pgsql -s 4326 -I "soil_type/全国土纲土亚纲土类.shp" public.soil_type | psql -d field_survey_db -U postgres
    shp2pgsql -s 4326 -I "sheet_name_num/sheet_name_num.shp" public.sheet_name_num | psql -d field_survey_db -U postgres


#### raster2pgsql -s 4326 -I -C -M -t 256x256 precipitation.tif public.precipitation | psql -d field_survey_db -U postgres

    raster2pgsql -s 4326 -I -C -M -t 256x256 "90m_DEM/全国90米dem高程.tif" public.dem_90m | psql -d field_survey_db -U postgres
    raster2pgsql -s 4326 -I -C -M -t 256x256 "landform_type/全国地貌类型.tif" public.landform_type | psql -d field_survey_db -U postgres
    raster2pgsql -s 4326 -I -C -M -t 256x256 "precipitation/precipitation.tif" public.precipitation | psql -d field_survey_db -U postgres
    raster2pgsql -s 4326 -I -C -M -t 512x512 "slope/全国90米坡度16位.tif" public.slope | psql -d field_survey_db -U postgres
    raster2pgsql -s 4326 -I -C -M -t 512x512 "aspect/全国90米坡向16位.tif" public.aspect | psql -d field_survey_db -U postgres

    *太大,暂时不上传*  raster2pgsql -s 4326 -I -C -M -t 512x512 "vegetation/全国植被型组.tif" public.vegetation | psql -d field_survey_db -U postgres

## Gin 的标准访问日志

journalctl -u FS_Backend.service -f

    INSERT INTO users (username, email, password, phone, role, created_at, last_login_date, last_ip, creator_id) VALUES
    ('一级项目负责人1', '@test.com', '$2a$10$oVamJtMxRBKQfizEoSPEqexuFoNtE.vU/.ilZaoTIJ110jYaJz4CC', '13800000001', 'first_admin', NOW(), NOW(), '127.0.0.1', 0),
    ('一级项目负责人2', '@test.com', '$2a$10$oVamJtMxRBKQfizEoSPEqexuFoNtE.vU/.ilZaoTIJ110jYaJz4CC', '13800000002', 'first_admin', NOW(), NOW(), '127.0.0.1', 0);

## 0417演示sql
INSERT INTO workspaces (project_id, assignee_id, name, description, file_url) VALUES
(1,152,'演示三级项目0417','演示','file');
