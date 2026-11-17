-- One database per service: services never read each other's tables.
-- Runs only the first time the postgres volume is created.
CREATE DATABASE auth;
CREATE DATABASE catalog;
CREATE DATABASE inventory;
CREATE DATABASE orders;
CREATE DATABASE payment;
-- Throwaway database for integration tests (never used by a running service).
CREATE DATABASE catalog_test;

-- Throwaway database for integration tests.
CREATE DATABASE auth_test;

-- Throwaway databases for integration tests.
CREATE DATABASE inventory_test;
CREATE DATABASE orders_test;

-- Throwaway database for the shared library tests.
CREATE DATABASE shared_test;

-- Throwaway database for payment integration tests.
CREATE DATABASE payment_test;
