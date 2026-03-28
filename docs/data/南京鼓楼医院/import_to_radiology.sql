-- 鼓楼医院旧库数据导入到当前系统（目标节点：放射科 dept-cardiology）
-- 范围：employees / employee_groups / shifts / shift_groups / shift_weekly_staff
-- 来源文件：docs/data/南京鼓楼医院/old.sql
-- 执行前提：当前库已存在组织节点 dept-cardiology（放射科）
-- 旧库不包含 group_members / fixed_assignments / shift_groups 明细数据，
-- 因此本脚本只导入主数据，并额外按“班次名=分组名”自动建立一批可推断的 shift_groups。
-- 执行方式：在当前系统数据库中直接执行

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

START TRANSACTION;

SET @target_org_node_id := 'dept-cardiology';
SET @target_org_node_name := '放射科';

-- 0. 前置校验：确认目标节点存在
SELECT id, name, node_type, path
FROM org_nodes
WHERE id = @target_org_node_id;

-- 1. 导入员工
-- 旧库映射：
-- org_id -> 丢弃，统一改为 org_node_id = dept-cardiology
-- employee_id -> employee_no
-- role -> category（旧数据当前为空）
-- hire_date(datetime) -> hire_date(YYYY-MM-DD)
-- 统一 scheduling_role=employee，app_must_reset_pwd=TRUE
INSERT INTO employees (
    id,
    org_node_id,
    name,
    employee_no,
    phone,
    email,
    position,
    category,
    scheduling_role,
    app_password_hash,
    app_must_reset_pwd,
    status,
    hire_date,
    created_at,
    updated_at
) VALUES
('010532f5-688f-4d21-804c-91b35e9dc6e7', @target_org_node_id, '孙双双', '062', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:37', '2025-11-06 14:45:37'),
('08a75281-0b60-4bdf-92d6-6463d1d61120', @target_org_node_id, '陈钱', '075', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:37', '2025-11-06 14:46:37'),
('0911e1e9-095f-424b-b387-11b595aa65d9', @target_org_node_id, '雷艳', '085', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:25', '2025-11-06 14:47:25'),
('0ac4522f-be27-4395-a118-38e6f1bb9745', @target_org_node_id, '周飞', '036', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:00', '2025-11-06 14:43:00'),
('0c02009a-bc33-41e9-8be8-1296dedcb3c0', @target_org_node_id, '王晟宇', '142', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:10', '2025-11-06 14:52:10'),
('0e30b5ba-33c3-4b9c-9126-7c86ddadde5e', @target_org_node_id, '解远卓', '140', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:02', '2025-11-06 14:52:02'),
('11f22c11-7b3b-4c41-8a22-1f14f22bd1ae', @target_org_node_id, '程涵', '061', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:33', '2025-11-06 14:45:33'),
('1459996f-08e3-42a3-8f2c-4e8d70833085', @target_org_node_id, '郐祥元', '115', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:50', '2025-11-06 14:49:50'),
('1763bf84-421e-4928-b5e8-e4f3cf62aaf9', @target_org_node_id, '董卉妍', '074', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:32', '2025-11-06 14:46:32'),
('1b887f9c-8f48-4686-81ea-dc66c56f4f29', @target_org_node_id, '张艳秋', '045', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:10', '2025-11-06 14:44:10'),
('1c21f4df-5e2b-4a53-900e-726dde03efe3', @target_org_node_id, '严陈晨', '027', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:50', '2025-11-06 14:40:50'),
('212f02a2-8b50-46d5-bc39-02ffb6a18140', @target_org_node_id, '陆加明', '050', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:38', '2025-11-06 14:44:38'),
('21fe9a5d-8a33-4ebd-b451-4e7327b15c30', @target_org_node_id, '辛睿静', '060', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:29', '2025-11-06 14:45:29'),
('26891585-3e15-40ec-b21c-88a9369a9e5a', @target_org_node_id, '傅琳清', '097', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:23', '2025-11-06 14:48:23'),
('27bf2d56-66c7-4669-a0cf-ed144eef4a58', @target_org_node_id, '王正阁', '021', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:21', '2025-11-06 14:40:21'),
('28ef5124-e57c-4ad4-8173-2c1efbd029fe', @target_org_node_id, '王浩丞', '092', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:59', '2025-11-06 14:47:59'),
('29202bd0-8ba3-4680-bb02-9e23d8ae511f', @target_org_node_id, '刘钰', '120', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:17', '2025-11-06 14:50:17'),
('2c4f62ff-6a5f-4b88-a8de-da5dc6288096', @target_org_node_id, '尹佳妮', '079', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:58', '2025-11-06 14:46:58'),
('2f14a9bd-f101-4362-9457-f252a19eeee2', @target_org_node_id, '孔翔', '146', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:31', '2025-11-06 14:52:31'),
('2fe01947-c733-4192-9e0e-396010371c8a', @target_org_node_id, '王凤仙', '054', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:00', '2025-11-06 14:45:00'),
('31274621-aef3-4977-9d1e-61b5a9736cc1', @target_org_node_id, '胡杰', '084', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:21', '2025-11-06 14:47:21'),
('33e10c96-f3c4-4358-9105-5be249167739', @target_org_node_id, '刘舒寒', '088', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:39', '2025-11-06 14:47:39'),
('3514386b-3e9d-4302-beba-013ca66c75aa', @target_org_node_id, '路芳逸', '149', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:46', '2025-11-06 16:53:02'),
('399a34d3-a753-4836-82b5-9516a4f7a21b', @target_org_node_id, '郭惠宁', '105', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:02', '2025-11-06 14:49:02'),
('3b43b7ef-b7a4-44f1-bac8-b40ff8dd79f2', @target_org_node_id, '陈静', '053', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:55', '2025-11-06 14:44:55'),
('3cbcd618-7d82-49a6-9aca-1d3c5db40de2', @target_org_node_id, '陆彤', '118', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:04', '2025-11-06 14:50:04'),
('3d891aab-17e2-4ecd-8cd5-1b4b77dbe977', @target_org_node_id, '韦贝贝', '137', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:49', '2025-11-06 14:51:49'),
('3e956831-4351-4557-bec0-031f6c5f3364', @target_org_node_id, '孙开波', '090', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:48', '2025-11-06 14:47:48'),
('3fc7d031-2955-4359-abed-2a7cbdfcc13d', @target_org_node_id, '樊祥红', '109', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:19', '2025-11-06 14:49:19'),
('42143147-8dd7-4b93-ba00-88636b54e38d', @target_org_node_id, '徐超', '128', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:11', '2025-11-06 14:51:11'),
('43aab6ac-be75-4270-a014-ea3eddf5780d', @target_org_node_id, '施佳倩', '069', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:08', '2025-11-06 14:46:08'),
('43ad3443-c016-446e-8a22-1da9ec062a86', @target_org_node_id, '田慧花', '134', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:36', '2025-11-06 14:51:36'),
('45d25180-f121-4616-b817-b06527989bca', @target_org_node_id, '孟婕', '048', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:28', '2025-11-06 14:44:28'),
('47783abe-15df-45e3-9dfc-34099139e6f9', @target_org_node_id, '刘子强', '147', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:36', '2025-11-06 14:52:36'),
('4876a61e-2e18-419f-9bf4-6e868180da13', @target_org_node_id, '吴嫚玮', '123', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:31', '2025-11-06 14:50:31'),
('4a22ea2c-35e6-477e-b4a1-c0e415000598', @target_org_node_id, '李扬', '052', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:49', '2025-11-06 14:44:49'),
('4ad60582-34ff-4af6-82f6-5af40cc46ec3', @target_org_node_id, '辛小燕', '008', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:36:58', '2025-11-06 14:36:58'),
('4c8306ad-aba3-470f-a3ab-492c816415f3', @target_org_node_id, '冯倩倩', '057', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:14', '2025-11-06 14:45:14'),
('4d0ee1ed-fabd-44d8-af3b-f1710da697b2', @target_org_node_id, '孙晶爽', '103', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:53', '2025-11-06 14:48:53'),
('4e748c63-bbdf-4ef0-af21-d2a9b1f1c1c0', @target_org_node_id, '吴羽彤', '122', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:27', '2025-11-06 14:50:27'),
('4fdb3b92-6291-4eef-b33b-0a10fe9caee9', @target_org_node_id, '徐晶', '089', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:43', '2025-11-06 14:47:43'),
('50a6ed32-395d-45dc-9a2e-05961e48c069', @target_org_node_id, '张海龙', '038', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:15', '2025-11-06 14:43:15'),
('5206f1b6-22d6-4db2-83c0-c2fa6b954e08', @target_org_node_id, '顾康康', '006', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:36:42', '2025-11-06 14:36:42'),
('5322ca36-6d31-48d3-921b-1a33b9a3f8c0', @target_org_node_id, '龙聪', '096', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:18', '2025-11-06 14:48:18'),
('538d3da3-a126-4e92-9a03-516060452155', @target_org_node_id, '陈伯柱', '020', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:15', '2025-11-06 14:40:15'),
('546101c7-cda2-434e-8555-dc788a9820ff', @target_org_node_id, '郑欢欢', '049', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:34', '2025-11-06 14:44:34'),
('572c7c54-07d3-4fc6-8e4c-e21d2fb502e1', @target_org_node_id, '姜星芝', '121', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:23', '2025-11-06 14:50:23'),
('5918747b-cfa6-47bc-a8ae-2758d1927fad', @target_org_node_id, '李冠', '015', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:43', '2025-11-06 14:37:43'),
('592b5c71-2b00-4400-9e6b-88a54c0f6a62', @target_org_node_id, '高娟娟', '136', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:45', '2025-11-06 14:51:45'),
('5c5f13ab-2f90-47ff-b801-43325cfcba49', @target_org_node_id, '孙梓婷', '129', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:15', '2025-11-06 14:51:15'),
('5c671b46-a14c-4bcb-969e-a82e151c5053', @target_org_node_id, '李姝影', '086', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:29', '2025-11-06 14:47:29'),
('5df8a8cb-8550-47d2-b2d1-fcb6275b1b2a', @target_org_node_id, '苏彤', '145', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:26', '2025-11-06 14:52:26'),
('5e57ef40-97f1-484a-8254-e9948431db62', @target_org_node_id, '梁静', '028', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:56', '2025-11-06 14:40:56'),
('5e68bcf6-fd9a-405f-ae61-4392c6be5387', @target_org_node_id, '李丹燕', '018', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:38:15', '2025-11-06 14:38:15'),
('5fb2603a-9904-4171-8b06-982df153a69c', @target_org_node_id, '王成', '080', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:03', '2025-11-06 14:47:03'),
('6290cf98-e61d-4db0-84d3-f22ee7b13b89', @target_org_node_id, '贺典', '144', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:19', '2025-11-06 14:52:19'),
('664c0b28-b15d-4f07-8db6-e45deaf2cf59', @target_org_node_id, '常莹', '013', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:29', '2025-11-06 14:37:29'),
('6960e06a-86e8-468f-9fc6-a4f783422676', @target_org_node_id, '张茂霖', '102', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:49', '2025-11-06 14:48:49'),
('6a972e61-9189-4158-8059-bb71c0d4fa36', @target_org_node_id, '张聪', '124', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:36', '2025-11-06 14:50:36'),
('6aae120f-4f5f-4506-afb6-c2d4f59b4df0', @target_org_node_id, '项蕾', '032', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:15', '2025-11-06 14:41:15'),
('6e8b1693-704f-410b-82a7-55d4e0f0dd99', @target_org_node_id, '戚荣丰', '016', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:59', '2025-11-06 14:37:59'),
('700b386d-de9b-4085-9db5-ea618a9c23dd', @target_org_node_id, '李如画', '138', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:53', '2025-11-06 14:51:53'),
('71687b13-a03a-4849-b2de-24bf98af5b67', @target_org_node_id, '乔婷婷', '101', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:45', '2025-11-06 14:48:45'),
('73a5fc2e-24f8-40c7-b34e-473ac0149a80', @target_org_node_id, '王军霞', '056', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:10', '2025-11-06 14:45:10'),
('7561e9df-5e88-442a-8ec1-a75673fe2993', @target_org_node_id, '朱雅静', '098', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:28', '2025-11-06 14:48:28'),
('794476b9-083e-423e-ab86-a3bcf86a4785', @target_org_node_id, '马义', '030', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:05', '2025-11-06 14:41:05'),
('7964872a-37ad-43a6-ae77-043faf979664', @target_org_node_id, '翟学', '112', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:33', '2025-11-06 14:49:33'),
('7abdfcf9-88da-4b3a-b452-13a9c612daaa', @target_org_node_id, '刘高平', '070', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:13', '2025-11-06 14:46:13'),
('7ce3ccda-f074-419b-9bf1-2b9fd3958e8a', @target_org_node_id, '令潇', '133', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:33', '2025-11-06 14:51:33'),
('7cf1dc16-2693-4656-877a-b66329f3482c', @target_org_node_id, '杨献峰', '009', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:05', '2025-11-06 14:37:05'),
('7d0529ff-d61e-49d3-bf54-4957921fbc22', @target_org_node_id, '吴德尚', '135', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:41', '2025-11-06 14:51:41'),
('7f47a28d-2293-459e-b49f-98afe56f72e6', @target_org_node_id, '刘丹', '087', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:35', '2025-11-06 14:47:35'),
('7ff7c03d-3da2-4046-9618-3df672ab1bed', @target_org_node_id, '杨硕', '130', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:20', '2025-11-06 14:51:20'),
('81c0b529-eeb5-477f-8296-ef302af9d77c', @target_org_node_id, '胡俊', '059', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:23', '2025-11-06 14:45:23'),
('84f6130e-2114-4b69-ba41-e8396e38cd43', @target_org_node_id, '梁雪', '031', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:10', '2025-11-06 14:41:10'),
('8599dab1-66d1-46be-83f0-6c75ac18a166', @target_org_node_id, '秦国初', '004', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 13:43:34', '2025-11-06 13:43:34'),
('85a2df6d-9de1-4428-8164-7ed95bfb1fd9', @target_org_node_id, '张昭奉', '148', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:41', '2025-11-06 16:23:53'),
('8720fab0-5568-4fba-87b5-e83a2bb40df0', @target_org_node_id, '李辉', '019', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:10', '2025-11-06 14:40:10'),
('8859162c-65db-439d-818e-ca7e27e91dfe', @target_org_node_id, '彭昕', '063', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:42', '2025-11-06 14:45:42'),
('8ae74911-4452-49d6-8717-175e7bb35577', @target_org_node_id, '佟琪', '026', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:47', '2025-11-06 14:40:47'),
('8b3d4791-7deb-4432-b916-7fe7973b16ef', @target_org_node_id, '李冬梅', '131', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:25', '2025-11-06 14:51:25'),
('8ccda435-15aa-4e82-a828-6e296a4aa305', @target_org_node_id, '黄桐宇', '139', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:57', '2025-11-06 14:51:57'),
('8de9b7ba-a942-40a7-a877-f51bb5f51459', @target_org_node_id, '梁欢欢', '095', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:11', '2025-11-06 14:48:11'),
('8debce96-db21-4218-9f99-fd524caac9b9', @target_org_node_id, '李旭晓', '117', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:00', '2025-11-06 14:50:00'),
('8df441b1-e359-486c-8334-1556807aa82d', @target_org_node_id, '陈夫涛', '082', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:12', '2025-11-06 14:47:12'),
('8e5c97bf-206c-4d12-b75a-2030bd2bb697', @target_org_node_id, '何雪颖', '064', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:46', '2025-11-06 14:45:46'),
('8f340693-01df-410d-9320-24aedec8049a', @target_org_node_id, '胡清', '073', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:28', '2025-11-06 14:46:28'),
('8f38854b-d271-4307-9d7c-ed1c2db4567f', @target_org_node_id, '储晨', '055', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:05', '2025-11-06 14:45:05'),
('983c6b82-95d2-4b17-bb49-382a77291747', @target_org_node_id, '余鸿鸣', '022', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:27', '2025-11-06 14:40:27'),
('9b0bbcbe-8fa6-4589-bad0-b0f1be75e2e4', @target_org_node_id, '魏晓磊', '025', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:42', '2025-11-06 14:40:42'),
('9d043735-f0f1-40a0-81c1-a85be6eae01d', @target_org_node_id, '宋雅琪', '100', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:40', '2025-11-06 14:48:40'),
('9d192e0a-93b2-451d-ad47-e4ea32cc4cca', @target_org_node_id, '杨惠泉', '068', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:03', '2025-11-06 14:46:03'),
('9e3b1115-3078-4f80-9b90-599bdc4fd798', @target_org_node_id, '陈玉灿', '083', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:16', '2025-11-06 14:47:16'),
('9e9bb8dd-f587-4ecb-afba-28f47fff69ef', @target_org_node_id, '叶梅萍', '039', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:20', '2025-11-06 14:43:20'),
('a262b6b2-6a7a-4b75-957c-42e61a6f45b9', @target_org_node_id, '周正扬', '003', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 13:43:23', '2025-11-06 13:43:23'),
('a334bcf3-f34c-4ab3-b161-9b9ccdd7f2f9', @target_org_node_id, '薛秋苍', '066', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:55', '2025-11-06 14:45:55'),
('a614e672-bee4-48d7-b7b9-fee136b4a78c', @target_org_node_id, '唐敏', '014', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:35', '2025-11-06 14:37:35'),
('a639ee57-5b6c-4c29-ae68-e5e8a2ab095a', @target_org_node_id, '季长风', '047', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:24', '2025-11-06 14:44:24'),
('a6f929a5-a4f9-4807-9c62-3ee036746743', @target_org_node_id, '麦筱莉', '007', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:36:49', '2025-11-06 14:36:49'),
('a9497e77-9709-4fc4-b942-c3809a4da87e', @target_org_node_id, '梅金婷', '111', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:29', '2025-11-06 14:49:29'),
('a9c1c578-f82c-41bd-b902-448bac11e3e1', @target_org_node_id, '张鑫', '010', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:12', '2025-11-06 14:37:12'),
('ac715ceb-7e4f-44d3-b0a5-48e87992728a', @target_org_node_id, '尹克杰', '058', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:19', '2025-11-06 14:45:19'),
('ad76924c-ea3a-4905-b0ef-5bbbbc62ee80', @target_org_node_id, '施婷婷', '017', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:38:08', '2025-11-06 14:38:08'),
('adcb5b71-1e55-4df3-bd2c-5f29b1482657', @target_org_node_id, '乔娜娜', '106', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:06', '2025-11-06 14:49:06'),
('aea36cec-80f0-42e9-85f7-2d66d99c8c7e', @target_org_node_id, '张易', '093', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:03', '2025-11-06 14:48:03'),
('b1892387-58e0-4ad8-950f-b78649229b3c', @target_org_node_id, '张雯', '076', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:42', '2025-11-06 14:46:42'),
('b2d9f34b-1481-4bc8-991b-d8e87af7e544', @target_org_node_id, '宫雯莉', '125', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:58', '2025-11-06 14:50:58'),
('b5506905-eee6-43df-a073-783ac1319881', @target_org_node_id, '王欢欢', '033', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:20', '2025-11-06 14:41:20'),
('b625ba9b-a2f7-44af-9e4d-a2b2d29850db', @target_org_node_id, '王晨', '072', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:23', '2025-11-06 14:46:23'),
('b7711633-784c-495f-b315-35409458a689', @target_org_node_id, '周晋', '035', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:31', '2025-11-06 14:41:31'),
('b7bf4789-e11d-4a5a-8f87-69afe7323f6f', @target_org_node_id, '祝丽', '043', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:01', '2025-11-06 14:44:01'),
('b85174fe-8d84-406d-8b42-73bba135fa5d', @target_org_node_id, '徐嘉佳', '108', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:15', '2025-11-06 14:49:15'),
('b86c32b9-91e4-44de-a1b3-6dc80a87a486', @target_org_node_id, '顾燕', '078', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:53', '2025-11-06 14:46:53'),
('ba17a21a-546d-4871-925d-aae9205869c9', @target_org_node_id, '叶粟', '116', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:55', '2025-11-06 14:49:55'),
('ba5a1569-f527-461b-b7d2-5992251b871e', @target_org_node_id, '丁月健', '141', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:07', '2025-11-06 14:52:07'),
('bbee1eaa-62d8-4db7-a635-2e18b5931a0c', @target_org_node_id, '李卫萍', '037', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:09', '2025-11-06 14:43:09'),
('c002f57b-e041-464c-8f29-f04597e8f124', @target_org_node_id, '窦鑫', '012', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:24', '2025-11-06 14:37:24'),
('c07d6ee9-5b9a-42b6-a525-9f9dad59fd08', @target_org_node_id, '李宝新', '002', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 13:43:17', '2025-11-06 13:43:17'),
('c27de258-0f50-44a8-85f5-f4cc0a408142', @target_org_node_id, '周楠', '046', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:15', '2025-11-06 14:44:15'),
('c2dd702b-fc75-4ce1-bde0-8b7fa5b40848', @target_org_node_id, '季耀', '126', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:03', '2025-11-06 14:51:03'),
('c74caf40-d0ce-4fad-b72e-ef24aad5d3b8', @target_org_node_id, '马林', '107', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:10', '2025-11-06 14:49:10'),
('c7d0c2c8-3ac2-41d9-91c3-be3c12b811a1', @target_org_node_id, '章志伟', '113', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:37', '2025-11-06 14:49:37'),
('c8c90a41-ab52-4e2b-97cd-04f97cead6dd', @target_org_node_id, '李欣', '081', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:08', '2025-11-06 14:47:08'),
('d2b4873a-e907-45b0-b107-9b845727a713', @target_org_node_id, '陈思璇', '040', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:29', '2025-11-06 14:43:29'),
('d4e28505-4aa0-457e-8817-ae166d367de4', @target_org_node_id, '张莉', '005', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:36:33', '2025-11-06 14:36:33'),
('d70aa76e-75da-40ad-b23a-585c45a6223c', @target_org_node_id, '祁宇', '071', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:18', '2025-11-06 14:46:18'),
('dc046678-fc1f-4307-a432-4157ee0852fc', @target_org_node_id, '施桦', '051', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:43', '2025-11-06 14:44:43'),
('dd9efcea-4abe-4079-b349-0cbb5f934643', @target_org_node_id, '武文博', '034', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:25', '2025-11-06 14:41:25'),
('de40160c-bc88-4897-9e5a-1ada53a6639d', @target_org_node_id, '徐梦颖', '094', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:07', '2025-11-06 14:48:07'),
('e0ab0721-ae2a-4138-95f3-b9ffb9d056ac', @target_org_node_id, '杨飞', '114', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:42', '2025-11-06 14:49:42'),
('e411b696-a34a-40b9-8381-6567f55f497b', @target_org_node_id, '刘宇', '044', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:44:05', '2025-11-06 14:44:05'),
('e438f171-baac-4033-9e4a-47d865536d2a', @target_org_node_id, '朱斌', '001', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 13:43:09', '2025-11-06 13:43:09'),
('e4fd4774-fa3c-49dd-8842-2f71cffa6c61', @target_org_node_id, '陈文萍', '029', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:41:01', '2025-11-06 14:41:01'),
('ea36ab46-0c39-4e34-8e5a-cf184939802c', @target_org_node_id, '张舒', '110', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:49:24', '2025-11-06 14:49:24'),
('eabb3e52-b3ad-49b4-a579-7599c54bef57', @target_org_node_id, '倪玲', '023', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:31', '2025-11-06 14:40:31'),
('eaf32e34-4223-4c68-9ddb-ebc173ddc28d', @target_org_node_id, '范沈豫', '091', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:47:55', '2025-11-06 14:47:55'),
('eaf9932d-bd31-43bc-9ef3-37af0fa3d8f9', @target_org_node_id, '邓启明', '104', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:57', '2025-11-06 14:48:57'),
('eebeb4cd-a23f-401b-bd42-baac730fbfc6', @target_org_node_id, '王茂雪', '067', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:59', '2025-11-06 14:45:59'),
('ef18fa7f-dc55-4d2c-9b3f-35b674ad236c', @target_org_node_id, '周群', '041', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:41', '2025-11-06 14:43:41'),
('ef9ce626-43bf-4da4-accf-c4f94d902801', @target_org_node_id, '杨雯', '042', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:43:56', '2025-11-06 14:43:56'),
('f0d3c7d3-f494-4161-9722-80abd26689fb', @target_org_node_id, '丁升', '065', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:45:51', '2025-11-06 14:45:51'),
('f0f80cba-4364-4af9-ba41-0978a7112eb2', @target_org_node_id, '张雪', '099', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:48:35', '2025-11-06 14:48:35'),
('f10df04b-b6dd-4261-a5e3-d15658a7956e', @target_org_node_id, '程乐', '077', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:46:47', '2025-11-06 14:46:47'),
('f52003e1-f12d-425e-b670-ce605f271bca', @target_org_node_id, '宋扬', '127', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:07', '2025-11-06 14:51:07'),
('f5e03ef7-bf23-40c5-8fc7-8a765476f486', @target_org_node_id, '袁芳', '132', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:51:29', '2025-11-06 14:51:29'),
('f5f381f3-5963-421e-85df-c4e2ba2e6c93', @target_org_node_id, '姜玥', '143', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:52:14', '2025-11-06 14:52:14'),
('f6e96ea8-64cd-4452-80fc-7a9bebae7db8', @target_org_node_id, '刘松', '024', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:40:37', '2025-11-06 14:40:37'),
('fa55bb2a-58e2-4acf-805b-9be531acabe9', @target_org_node_id, '孙晓敏', '011', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:37:19', '2025-11-06 14:37:19'),
('fcb3fee9-fcb3-460c-92d2-eef681d8fa8d', @target_org_node_id, '王子晗', '119', NULL, NULL, NULL, NULL, 'employee', NULL, TRUE, 'active', NULL, '2025-11-06 14:50:10', '2025-11-06 14:50:10')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    phone = VALUES(phone),
    email = VALUES(email),
    position = VALUES(position),
    category = VALUES(category),
    status = VALUES(status),
    hire_date = VALUES(hire_date),
    updated_at = VALUES(updated_at);

-- 2. 导入分组
-- 旧 groups.code / type / attributes / leader_id / parent_id 在当前系统无直接落点
-- 暂存到 description 中，避免信息丢失
-- 注意：由于 old.sql 没有员工-分组成员关系表，这里不会写 group_members
INSERT INTO employee_groups (
    id,
    org_node_id,
    name,
    description,
    created_at,
    updated_at
) VALUES
('05a3bea2-2b9f-4ed4-84de-4e5b556cbe5f', @target_org_node_id, 'DR审核组', 'old_code=104; old_type=team', '2025-11-06 20:08:00', '2025-11-06 20:08:00'),
('2a009e64-7eab-4f32-bd81-5ef96d6551bc', @target_org_node_id, '审核/报告组', 'old_code=103; old_type=team', '2025-11-06 15:19:19', '2025-11-06 15:19:19'),
('4910c762-2f39-483e-9aaf-e7204d330331', @target_org_node_id, '急诊审核', 'old_code=209; old_type=team', '2025-12-16 10:33:52', '2025-12-16 10:41:41'),
('58464a49-80e8-438d-8e5d-8623523ece60', @target_org_node_id, 'CT/MRI轮转1', 'old_code=202; old_type=team', '2025-11-08 09:10:51', '2025-11-08 09:10:51'),
('59c228ea-102a-4ade-aa2c-f64ba7b60a57', @target_org_node_id, '江北夜班', 'old_code=303; old_type=team', '2025-11-07 09:30:20', '2025-11-07 09:30:20'),
('86a00075-25e3-4afb-b509-81ecf228d20f', @target_org_node_id, '科研班', 'old_code=301; old_type=team', '2025-11-08 09:19:54', '2025-11-08 09:19:54'),
('9c2e3ab0-fa81-4266-aa75-4a694ea268b8', @target_org_node_id, 'CT/MRI轮转2', 'old_code=203; old_type=team', '2025-11-08 09:18:21', '2025-11-08 09:18:21'),
('a1727d63-7cbe-4287-9430-acc218e50118', @target_org_node_id, '非固定审核组', 'old_code=102; old_type=team', '2025-11-06 15:18:17', '2025-11-06 15:18:17'),
('add1015e-5f24-4773-9ae3-ccf5d42ee749', @target_org_node_id, '职工报告组', 'old_code=204; old_type=team', '2025-11-06 15:20:01', '2025-12-16 10:18:12'),
('b2528c69-fa24-44ee-9808-dafa0a88edb4', @target_org_node_id, '江北穿刺', 'old_code=205; old_type=team', '2025-11-08 09:02:41', '2025-12-16 11:06:03'),
('c4837ec9-bfbd-4be6-8d90-dff214d80d0f', @target_org_node_id, '本部夜班', 'old_code=302; old_type=team', '2025-11-08 09:05:57', '2025-11-08 09:05:57'),
('c48954ba-dd28-4e5b-935b-8438c145932d', @target_org_node_id, '光子冠脉CTA', 'old_code=206; old_type=team', '2025-11-08 09:00:55', '2025-12-31 11:48:30'),
('d0b0ecff-ed63-4842-887a-153d80fac274', @target_org_node_id, 'DR学员轮转组', '定期轮转; old_code=201; old_type=team', '2025-11-06 20:03:55', '2025-11-06 20:05:31'),
('d43f00bf-7ad0-4b77-9b94-ce5c9ade2f22', @target_org_node_id, '急诊审核', 'old_code=105; old_type=team', '2025-11-08 09:24:50', '2025-11-08 09:24:50'),
('e0f7950e-f3e6-42e3-b867-f4857d2e7c27', @target_org_node_id, '新入职', 'old_code=207; old_type=team', '2025-11-08 09:08:38', '2025-11-08 09:08:38'),
('e17cebec-6e6a-4478-af8f-bd6df8444a77', @target_org_node_id, '本部穿刺', 'old_code=210; old_type=team', '2025-12-16 11:07:13', '2025-12-16 11:07:13'),
('fcd069bf-9667-43b8-ab99-c424fc339c63', @target_org_node_id, '固定审核组', 'old_code=101; old_type=team', '2025-11-06 15:17:11', '2025-11-06 15:17:11')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    description = VALUES(description),
    updated_at = VALUES(updated_at);

-- 3. 导入班次
-- 旧库字段映射：
-- is_overnight -> is_cross_day
-- is_active -> status(active/disabled)
-- scheduling_priority -> priority
-- default_staff_count -> 后续写入 shift_weekly_staff
INSERT INTO shifts (
    id,
    org_node_id,
    name,
    code,
    type,
    description,
    start_time,
    end_time,
    duration,
    is_cross_day,
    color,
    priority,
    status,
    metadata,
    created_at,
    updated_at
) VALUES
('07912c61-f2d7-4e2b-a92e-2856c5ca859c', @target_org_node_id, 'CT/MRI审核上', '201', 'regular', 'CT/MRI审核报告-上午班次', '08:00', '14:00', 360, FALSE, '#67C23A', 113, 'active', JSON_OBJECT('legacy_default_staff_count', 17, 'legacy_priority', 0), '2025-11-06 15:14:26', '2026-03-13 11:34:40'),
('1eee1f6b-2439-4823-9ec0-d9c0f7c522fd', @target_org_node_id, 'DR报告下', '106', 'regular', 'DR写报告-下午班次', '13:30', '17:30', 240, FALSE, '#409EFF', 12, 'disabled', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-06 15:09:58', '2025-11-08 10:27:59'),
('217364fc-2b99-4aa7-9f5e-a1bc3e13657a', @target_org_node_id, 'CT/MRI审核下', '202', 'regular', 'CT/MRI审核报告-下午班次', '14:00', '20:00', 360, FALSE, '#409EFF', 114, 'active', JSON_OBJECT('legacy_default_staff_count', 17, 'legacy_priority', 0), '2025-11-06 15:14:49', '2026-03-13 14:13:35'),
('22223215-37cc-4a2b-aa31-705b1e5018d0', @target_org_node_id, '下夜班', '307', 'regular', '周一至周日', '00:00', '08:00', 480, FALSE, '#AA7E9A', 4, 'active', JSON_OBJECT('legacy_default_staff_count', 2, 'legacy_priority', 0), '2025-11-08 10:12:03', '2026-03-13 15:05:46'),
('256c06f9-235c-4d78-91f9-9228cc107c19', @target_org_node_id, '科研', '304', 'regular', '周一至周五', '08:00', '18:00', 600, FALSE, '#20B2AA', 104, 'disabled', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:06:20', '2025-12-24 16:28:58'),
('25afbbf1-0077-4ccb-a788-d311a8f99bc0', @target_org_node_id, '江北夜班', '306', 'regular', '周一至周日', '20:00', '24:00', 240, FALSE, '#E6A23C', 3, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:10:36', '2026-03-13 17:22:48'),
('3313cf99-b4f7-4116-8634-5e9a46a81e41', @target_org_node_id, '急诊审核下', '402', 'regular', '', '14:00', '20:00', 360, FALSE, '#FB688D', 8, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-12-31 10:47:53', '2026-03-13 11:34:28'),
('35897995-a027-48c4-86f5-7411b99f836e', @target_org_node_id, '江北穿刺', '302', 'regular', '不显示在排班表上', '14:00', '20:00', 360, FALSE, '#909399', 1, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:02:23', '2026-03-13 15:09:26'),
('4d4a1edb-15e6-4079-80b2-eb5fa59926e3', @target_org_node_id, '本部穿刺', '301', 'regular', '本部穿刺当天与其它班次互斥', '08:00', '20:00', 720, FALSE, '#C71585', 5, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:00:42', '2025-12-31 11:32:20'),
('5573cfa4-417d-432e-9c6f-77d2979ab689', @target_org_node_id, 'DR审核上', '203', 'regular', 'DR审核报告-上午班次', '08:00', '12:00', 240, FALSE, '#67C23A', 111, 'disabled', JSON_OBJECT('legacy_default_staff_count', 10, 'legacy_priority', 0), '2025-11-06 15:15:44', '2025-11-21 14:38:55'),
('5fb734b0-4929-4ff3-8f2b-9edbfc438954', @target_org_node_id, 'CT/MRI报告下', '108', 'regular', 'CT+MRI合班写报告-下午班次', '14:00', '20:00', 360, FALSE, '#409EFF', 116, 'active', JSON_OBJECT('legacy_default_staff_count', 9, 'legacy_priority', 0), '2025-11-06 15:11:34', '2026-03-13 15:10:54'),
('6147eb2d-d575-414a-9630-97fe5aca401b', @target_org_node_id, 'CT报告上', '101', 'regular', 'CT写报告-上午班次', '08:00', '14:00', 360, FALSE, '#67C23A', 13, 'disabled', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-06 14:56:52', '2025-12-24 16:28:54'),
('6b4cd747-dce1-4c57-a5c0-04db2c3c45b6', @target_org_node_id, '光子冠脉CTA', '303', 'regular', '周一至周五，一天一个', '14:00', '20:00', 360, FALSE, '#FF69B4', 2, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:04:11', '2026-03-13 17:27:13'),
('74a0c63a-a45d-42fc-984f-80ee31311452', @target_org_node_id, 'MRI报告下', '104', 'regular', 'MRI写报告-下午班次', '14:00', '20:00', 360, FALSE, '#409EFF', 16, 'disabled', JSON_OBJECT('legacy_default_staff_count', 4, 'legacy_priority', 0), '2025-11-06 15:07:03', '2025-12-24 16:28:56'),
('78e39561-0f57-4e25-8312-a3ca6ce12eda', @target_org_node_id, 'DR报告上', '105', 'regular', 'DR写报告-上午班次', '08:00', '12:00', 240, FALSE, '#67C23A', 11, 'disabled', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-06 15:09:19', '2025-11-08 10:27:58'),
('78f7e5e2-8e5f-4f04-b38f-e21e5ac57e27', @target_org_node_id, 'MRI报告上', '103', 'regular', 'MRI写报告-上午班次', '08:00', '14:00', 360, FALSE, '#67C23A', 15, 'disabled', JSON_OBJECT('legacy_default_staff_count', 5, 'legacy_priority', 0), '2025-11-06 15:02:09', '2025-12-26 13:42:29'),
('ab846727-01c6-4190-bef7-0d218b7dfeaa', @target_org_node_id, 'CT报告下', '102', 'regular', 'CT写报告-下午班次', '14:00', '20:00', 360, FALSE, '#409EFF', 14, 'disabled', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-06 15:00:55', '2025-12-24 16:28:55'),
('c803c325-dc3c-4738-b5f2-bb030089b7e2', @target_org_node_id, 'CT/MRI报告上', '107', 'regular', 'CT+MRI合班写报告-上午班次', '08:00', '14:00', 360, FALSE, '#67C23A', 115, 'active', JSON_OBJECT('legacy_default_staff_count', 22, 'legacy_priority', 0), '2025-11-06 15:10:54', '2026-03-13 15:10:47'),
('e2351a22-85db-4cc9-b3a2-dc27d5521007', @target_org_node_id, '急诊审核上', '401', 'regular', '', '08:00', '14:00', 360, FALSE, '#FB688D', 7, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-12-31 10:46:58', '2026-03-13 11:34:26'),
('ebfab16f-6326-46d0-8009-ecbc77f72fa5', @target_org_node_id, 'DR审核下', '204', 'regular', 'DR审核报告-下午班次', '13:30', '17:30', 240, FALSE, '#409EFF', 112, 'disabled', JSON_OBJECT('legacy_default_staff_count', 10, 'legacy_priority', 0), '2025-11-06 15:16:04', '2025-11-21 14:39:03'),
('f9ee2a80-678a-4453-b31d-e8ebe2d31ffb', @target_org_node_id, '本部夜班', '305', 'regular', '周一至周日', '20:00', '24:00', 240, FALSE, '#E6A23C', 3, 'active', JSON_OBJECT('legacy_default_staff_count', 1, 'legacy_priority', 0), '2025-11-08 10:08:55', '2026-03-13 17:23:01')
ON DUPLICATE KEY UPDATE
    name = VALUES(name),
    type = VALUES(type),
    description = VALUES(description),
    start_time = VALUES(start_time),
    end_time = VALUES(end_time),
    duration = VALUES(duration),
    is_cross_day = VALUES(is_cross_day),
    color = VALUES(color),
    priority = VALUES(priority),
    status = VALUES(status),
    metadata = VALUES(metadata),
    updated_at = VALUES(updated_at);

-- 4. 导入班次默认关联分组
-- 规则：按名称完全匹配自动建立关联；只覆盖旧库中同名的班次和分组
INSERT INTO shift_groups (
    id,
    org_node_id,
    shift_id,
    group_id,
    priority,
    is_active,
    notes,
    created_at,
    updated_at
)
SELECT
    CONCAT('sg-', s.id, '-', g.id) AS id,
    @target_org_node_id,
    s.id,
    g.id,
    0,
    TRUE,
    '按旧库名称自动匹配导入',
    NOW(),
    NOW()
FROM shifts s
JOIN employee_groups g
  ON s.org_node_id = g.org_node_id
 AND s.name = g.name
WHERE s.org_node_id = @target_org_node_id
ON DUPLICATE KEY UPDATE
    is_active = VALUES(is_active),
    notes = VALUES(notes),
    updated_at = VALUES(updated_at);

-- 5. 导入班次默认周人数配置
-- 旧库 default_staff_count 没有逐周几拆分，这里按“周一到周日统一人数”导入
INSERT INTO shift_weekly_staff (
    id,
    org_node_id,
    shift_id,
    weekday,
    staff_count,
    is_custom,
    created_at,
    updated_at
)
SELECT CONCAT('sws-', s.id, '-0'), @target_org_node_id, s.id, 0,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-1'), @target_org_node_id, s.id, 1,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-2'), @target_org_node_id, s.id, 2,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-3'), @target_org_node_id, s.id, 3,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-4'), @target_org_node_id, s.id, 4,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-5'), @target_org_node_id, s.id, 5,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
UNION ALL
SELECT CONCAT('sws-', s.id, '-6'), @target_org_node_id, s.id, 6,
       CAST(JSON_UNQUOTE(JSON_EXTRACT(s.metadata, '$.legacy_default_staff_count')) AS UNSIGNED),
       FALSE, NOW(), NOW()
FROM shifts s
WHERE s.org_node_id = @target_org_node_id
ON DUPLICATE KEY UPDATE
    staff_count = VALUES(staff_count),
    is_custom = VALUES(is_custom),
    updated_at = VALUES(updated_at);

COMMIT;

SET FOREIGN_KEY_CHECKS = 1;

-- 导入后建议核对：
-- 1. SELECT COUNT(*) FROM employees WHERE org_node_id = 'dept-cardiology';          预期 149
-- 2. SELECT COUNT(*) FROM employee_groups WHERE org_node_id = 'dept-cardiology';    预期 17
-- 3. SELECT COUNT(*) FROM shifts WHERE org_node_id = 'dept-cardiology';              预期 21
-- 4. SELECT s.name, g.name FROM shift_groups sg JOIN shifts s ON s.id = sg.shift_id JOIN employee_groups g ON g.id = sg.group_id WHERE sg.org_node_id = 'dept-cardiology';
