-- Development only: point Hospital A at the mock HIS container instead of the real API.
-- Mounted by docker-compose.yml after migrations/001_init.sql; never part of a real deployment.
UPDATE hospitals SET api_base_url = 'http://mock-his:8081' WHERE code = 'hospital-a';
