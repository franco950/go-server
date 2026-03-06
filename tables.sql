DROP TABLE IF EXISTS milk;
CREATE TABLE milk (
  id      INT AUTO_INCREMENT NOT NULL,
  cowid   VARCHAR(20) NOT NULL,
  fat     DECIMAL(4,2) NOT NULL,
  protein DECIMAL(4,2) NOT NULL,
  pH      DECIMAL(3,1) NOT NULL,
  scc     INT NOT NULL,
  PRIMARY KEY (`id`)
);

INSERT INTO milk
  (cowid, fat, protein, pH, scc)
VALUES
  ('COW-001', 3.8, 3.2, 6.6, 180000),
  ('COW-002', 4.1, 3.4, 6.7, 150000),
  ('COW-003', 3.5, 3.1, 6.5, 220000),
  ('COW-004', 4.3, 3.6, 6.8, 130000),
  ('COW-005', 3.9, 3.3, 6.6, 170000);