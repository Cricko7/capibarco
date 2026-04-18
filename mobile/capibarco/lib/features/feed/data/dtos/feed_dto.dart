import '../../domain/entities/feed_models.dart';

class FeedCardDto {
  const FeedCardDto({
    required this.id,
    required this.feedSessionId,
    required this.animalId,
    required this.name,
    required this.species,
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
  final String species;
  final String description;
  final String ownerDisplayName;
  final String photoUrl;
  final String city;
  final bool boosted;
  final List<String> rankingReasons;

  factory FeedCardDto.fromJson(Map<String, dynamic> json) {
    final animal =
        json['animal'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    final photos = animal['photos'] as List<dynamic>? ?? const <dynamic>[];
    final firstPhoto = photos.isNotEmpty
        ? photos.first as Map<String, dynamic>
        : const <String, dynamic>{};
    final location =
        animal['location'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return FeedCardDto(
      id: json['feed_card_id'] as String? ?? '',
      feedSessionId: json['feed_session_id'] as String? ?? '',
      animalId: animal['animal_id'] as String? ?? '',
      name: animal['name'] as String? ?? 'Unknown',
      species: animal['species'] as String? ?? 'SPECIES_UNSPECIFIED',
      description: animal['description'] as String? ?? '',
      ownerDisplayName: json['owner_display_name'] as String? ?? 'Caregiver',
      photoUrl: firstPhoto['url'] as String? ?? '',
      city: location['city'] as String? ?? '',
      boosted: json['boosted'] as bool? ?? false,
      rankingReasons:
          (json['ranking_reasons'] as List<dynamic>? ?? const <dynamic>[])
              .whereType<String>()
              .toList(),
    );
  }

  FeedCardEntity toDomain() {
    return FeedCardEntity(
      id: id,
      feedSessionId: feedSessionId,
      animalId: animalId,
      name: name,
      speciesLabel: species.replaceAll('SPECIES_', '').toLowerCase(),
      description: description,
      ownerDisplayName: ownerDisplayName,
      photoUrl: photoUrl,
      city: city,
      boosted: boosted,
      rankingReasons: rankingReasons,
    );
  }
}

class FeedPageDto {
  const FeedPageDto({
    required this.cards,
    required this.nextPageToken,
    required this.feedSessionId,
    this.isStale = false,
  });

  final List<FeedCardDto> cards;
  final String nextPageToken;
  final String feedSessionId;
  final bool isStale;

  factory FeedPageDto.fromJson(
    Map<String, dynamic> json, {
    bool isStale = false,
  }) {
    return FeedPageDto(
      cards: (json['cards'] as List<dynamic>? ?? const <dynamic>[])
          .map((item) => FeedCardDto.fromJson(item as Map<String, dynamic>))
          .toList(),
      nextPageToken: json['next_page_token'] as String? ?? '',
      feedSessionId: json['feed_session_id'] as String? ?? '',
      isStale: isStale,
    );
  }

  FeedPageEntity toDomain() {
    return FeedPageEntity(
      cards: cards.map((card) => card.toDomain()).toList(),
      nextPageToken: nextPageToken,
      feedSessionId: feedSessionId,
      isStale: isStale,
    );
  }
}
