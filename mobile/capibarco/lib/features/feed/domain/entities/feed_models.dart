class FeedCardEntity {
  const FeedCardEntity({
    required this.id,
    required this.feedSessionId,
    required this.animalId,
    required this.name,
    required this.speciesLabel,
    required this.description,
    required this.ownerDisplayName,
    required this.photoUrl,
    required this.city,
    required this.boosted,
    required this.rankingReasons,
  });

  final String id;
  final String feedSessionId;
  final String animalId;
  final String name;
  final String speciesLabel;
  final String description;
  final String ownerDisplayName;
  final String photoUrl;
  final String city;
  final bool boosted;
  final List<String> rankingReasons;
}

class FeedPageEntity {
  const FeedPageEntity({
    required this.cards,
    required this.nextPageToken,
    required this.feedSessionId,
    required this.isStale,
  });

  final List<FeedCardEntity> cards;
  final String nextPageToken;
  final String feedSessionId;
  final bool isStale;
}

class SwipeOutcome {
  const SwipeOutcome({required this.matched, this.conversationId});

  final bool matched;
  final String? conversationId;
}
