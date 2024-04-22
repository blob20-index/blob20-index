/*
 Navicat Premium Data Transfer

 Source Server         : local
 Source Server Type    : MySQL
 Source Server Version : 80033 (8.0.33)
 Source Host           : localhost:3306
 Source Schema         : blob27

 Target Server Type    : MySQL
 Target Server Version : 80033 (8.0.33)
 File Encoding         : 65001

 Date: 22/04/2024 20:00:58
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for blob20_account
-- ----------------------------
DROP TABLE IF EXISTS `blob20_account`;
CREATE TABLE `blob20_account` (
  `address` varchar(42) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `protocol` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `ticker` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `balance` decimal(18,8) unsigned NOT NULL DEFAULT '0.00000000',
  PRIMARY KEY (`address`,`protocol`,`ticker`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for blob20_deploy
-- ----------------------------
DROP TABLE IF EXISTS `blob20_deploy`;
CREATE TABLE `blob20_deploy` (
  `transaction_hash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `block_blockhash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `block_number` int NOT NULL,
  `block_timestamp` int NOT NULL,
  `deployer` varchar(42) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `protocol` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `ticker` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `max_supply` decimal(36,8) unsigned DEFAULT NULL,
  `max_limit_per_mint` decimal(18,8) unsigned DEFAULT NULL,
  `decimals` int DEFAULT NULL,
  `mint_amount` decimal(36,8) unsigned DEFAULT NULL,
  `mint_quantity` int DEFAULT NULL,
  `mint_start_block` int DEFAULT NULL,
  `mint_end_block` int DEFAULT NULL,
  PRIMARY KEY (`protocol`,`ticker`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for blob20_record
-- ----------------------------
DROP TABLE IF EXISTS `blob20_record`;
CREATE TABLE `blob20_record` (
  `transaction_hash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `block_blockhash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `block_number` int NOT NULL,
  `block_timestamp` int NOT NULL,
  `index` int NOT NULL,
  `protocol` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `ticker` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `operation` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `from` varchar(42) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `to` varchar(42) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `amount` decimal(18,8) NOT NULL,
  `from_before_amount` decimal(18,8) NOT NULL,
  `from_after_amount` decimal(18,8) NOT NULL,
  `to_before_amount` decimal(18,8) NOT NULL,
  `to_after_amount` decimal(18,8) NOT NULL,
  `gas_fee` decimal(36,18) DEFAULT NULL,
  `status` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `status_msg` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `remark` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  PRIMARY KEY (`transaction_hash`,`from`,`to`,`index`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Table structure for blob_transactions
-- ----------------------------
DROP TABLE IF EXISTS `blob_transactions`;
CREATE TABLE `blob_transactions` (
  `transaction_hash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
  `block_number` int DEFAULT NULL,
  `transaction_index` int DEFAULT NULL,
  `block_timestamp` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `block_blockhash` varchar(66) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `event_log_index` int DEFAULT NULL,
  `ethscription_number` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `creator` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `initial_owner` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `current_owner` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `previous_owner` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `content_uri` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `content_sha` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `esip6` tinyint(1) DEFAULT NULL,
  `mimetype` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `media_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `mime_subtype` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `gas_price` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `gas_used` int DEFAULT NULL,
  `transaction_fee` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `value` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `attachment_sha` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `attachment_content_type` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `attachment_path` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `blob20` text CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci,
  `blob_gas_price` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `blob_gas_used` int DEFAULT NULL,
  `blob_gas_fee` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `protocol` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `ticker` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `operation` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
  `amount` decimal(18,8) DEFAULT NULL,
  `is_valid` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`transaction_hash`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET FOREIGN_KEY_CHECKS = 1;
