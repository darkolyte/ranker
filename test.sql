SELECT 
    i_winner.name AS winner, 
    CASE 
        WHEN r.winner_id = r.first_item_id THEN i2.name 
        ELSE i1.name 
    END AS loser
FROM rankings r
JOIN items i1 ON r.first_item_id = i1.id
JOIN items i2 ON r.second_item_id = i2.id
JOIN items i_winner ON r.winner_id = i_winner.id
WHERE r.collection_id = 1;