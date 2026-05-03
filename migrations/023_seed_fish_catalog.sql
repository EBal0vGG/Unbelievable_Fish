INSERT INTO fish (id, name, description) VALUES
    ('fish-salmon-atlantic', 'Атлантический лосось', 'Базовая позиция для охлажденных и live B2B-поставок.'),
    ('fish-pollock-far-east', 'Минтай дальневосточный', 'Массовая белая рыба для заморозки, переработки и экспорта.'),
    ('fish-herring-north', 'Сельдь северная', 'Сырье для переработки, HoReCa и оптовых партий.'),
    ('fish-cod-atlantic', 'Атлантическая треска', 'Белая рыба для охлажденных и замороженных поставок.'),
    ('fish-mackerel-atlantic', 'Атлантическая скумбрия', 'Жирная морская рыба для заморозки и переработки.'),
    ('fish-haddock', 'Пикша', 'Белая рыба для филе, блоков и HoReCa.'),
    ('fish-trout-rainbow', 'Форель радужная', 'Аквакультура для охлажденных поставок и филе.'),
    ('fish-halibut-greenland', 'Палтус черный', 'Премиальная позиция для заморозки и филе.')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description;
