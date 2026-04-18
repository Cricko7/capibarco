import 'package:capibarco/features/feed/data/dtos/feed_dto.dart';
import 'package:capibarco/features/notifications/data/dtos/notification_dto.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('FeedCardDto accepts numeric enum payloads from gateway JSON', () {
    final dto = FeedCardDto.fromJson(<String, dynamic>{
      'feed_card_id': 'card-1',
      'feed_session_id': 'session-1',
      'animal': <String, dynamic>{
        'animal_id': 'animal-1',
        'owner_profile_id': 'owner-1',
        'name': 'Mila',
        'species': 2,
        'description': 'Friendly',
        'photos': <Map<String, dynamic>>[
          <String, dynamic>{'url': 'https://example.test/cat.jpg'},
        ],
        'location': <String, dynamic>{'city': 'Moscow'},
      },
      'owner_display_name': 'Shelter',
      'boosted': 1,
      'ranking_reasons': <dynamic>['recent', 42],
    });

    expect(dto.species, 'SPECIES_CAT');
    expect(dto.boosted, isTrue);
    expect(dto.rankingReasons, <String>['recent', '42']);
  });

  test('NotificationItemDto accepts numeric enums from proto-backed JSON', () {
    final dto = NotificationItemDto.fromJson(<String, dynamic>{
      'notification_id': 'notification-1',
      'title': 'New match',
      'body': 'You have a new match',
      'type': 1,
      'status': 2,
      'created_at': '2026-04-18T20:00:00Z',
      'read_at': null,
    });

    expect(dto.type, 'NOTIFICATION_TYPE_MATCH_CREATED');
    expect(dto.status, 'NOTIFICATION_STATUS_DELIVERED');
  });
}
