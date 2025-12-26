package seeders

import (
	"go-starter/internal/models"
	"math/rand"
	"strings"
	"time"
)

var loremIpsum = `
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer faucibus sapien lobortis consequat scelerisque. Duis at hendrerit dui. Nam laoreet diam vulputate lacus molestie ullamcorper. Suspendisse convallis lectus eget mauris lacinia vulputate. Quisque justo sem, tincidunt semper ullamcorper ut, tincidunt in nibh. Vestibulum finibus finibus metus, in sodales libero pretium vitae. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Maecenas et leo pellentesque, congue lorem eget, consectetur mi. Mauris id congue nisi. Proin non blandit lacus, a convallis neque. Vestibulum in malesuada libero.

Integer vitae bibendum nulla. Nunc nisl nisl, efficitur in sodales vel, pretium vitae eros. Fusce imperdiet sagittis tincidunt. Mauris sed dui sagittis libero mollis malesuada. Vestibulum maximus euismod velit nec dictum. Maecenas lobortis ultricies dapibus. Nulla id vestibulum turpis. Cras eu ante dolor.

Nam tempor mi vel purus porta posuere. Morbi massa leo, pretium et elit quis, pulvinar venenatis nisl. Praesent eleifend dolor eget diam faucibus, rutrum viverra mauris egestas. Fusce suscipit neque leo, sed elementum est ornare vitae. Nam interdum vehicula elementum. Nulla vestibulum semper venenatis. Suspendisse cursus erat ligula, nec dictum enim pellentesque ut.

Curabitur tincidunt nibh magna, vitae condimentum erat porttitor vel. Nullam nec tellus sed magna egestas vulputate at vitae tellus. Etiam vel tellus a erat finibus volutpat sagittis in est. Vestibulum euismod felis elementum nisl scelerisque, eget faucibus elit iaculis. Maecenas pretium ex augue, eget tempor sapien rhoncus eu. Vivamus vulputate lorem urna, feugiat tristique tellus hendrerit et. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Curabitur lobortis non dui sed porta.

Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Aliquam placerat auctor mauris, a blandit diam. Nulla aliquet lectus eget velit scelerisque vehicula. Sed ligula justo, fermentum et dictum eu, elementum eget lorem. Nunc sapien risus, convallis quis sem ut, mollis porttitor nibh. Pellentesque consequat molestie dui et aliquet. In sed pretium magna, in accumsan nunc. Etiam ut augue porttitor diam rhoncus pharetra id a nisi.

Aliquam laoreet luctus orci, sit amet elementum purus bibendum eget. Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Quisque lectus tortor, condimentum venenatis erat nec, dignissim aliquam urna. Praesent non ex id odio elementum luctus. Sed sodales ligula eu risus rutrum, non mattis diam euismod. Phasellus convallis eget ipsum non pellentesque. Maecenas vitae iaculis nisi. Phasellus et aliquam odio. Nam sit amet tristique neque. Ut turpis orci, rutrum vitae sem a, faucibus ullamcorper risus. Aenean nisi massa, elementum sed posuere nec, gravida auctor tellus. Nullam commodo dictum nisl, in malesuada orci accumsan vitae. Integer non velit tristique, aliquet tellus quis, laoreet ipsum.

Curabitur vel massa velit. Phasellus mattis maximus purus, tincidunt accumsan lacus sollicitudin non. Donec consectetur leo condimentum nibh viverra, vitae vehicula leo viverra. Donec varius pulvinar elit. Integer ac elementum diam. Mauris et sem porta, rhoncus dolor non, lobortis turpis. Aenean lobortis eros quis ullamcorper feugiat. Integer rhoncus tristique quam, mollis laoreet leo. Cras facilisis odio id ipsum varius dapibus. Morbi lobortis ipsum et risus egestas, eu condimentum libero elementum. Phasellus efficitur consequat nisl vitae ultrices. Donec auctor ligula nunc, in pharetra ex molestie at. Suspendisse arcu lectus, molestie tincidunt laoreet id, tincidunt ut enim.

Integer tempus augue vel risus venenatis, quis lobortis lorem placerat. Morbi dui nunc, pharetra ac condimentum sed, blandit congue neque. Suspendisse potenti. Vestibulum blandit lobortis gravida. Etiam vitae vulputate erat. Maecenas ultricies arcu et neque ullamcorper pellentesque in ut turpis. Donec id neque placerat, dapibus erat vitae, porta erat. Aliquam a lorem diam. Sed placerat metus blandit, placerat arcu ac, posuere libero. Maecenas pulvinar volutpat hendrerit. Sed eleifend turpis id augue pellentesque semper. Cras vel quam lectus. Nulla molestie aliquet malesuada.

Suspendisse eu purus viverra, convallis ante ut, aliquet est. Donec convallis velit at diam euismod laoreet. Curabitur id diam at justo pulvinar tincidunt et sit amet magna. Aliquam erat volutpat. Nullam in nisi pulvinar, lacinia elit vel, auctor dui. Proin efficitur augue quam, ut aliquam dolor lobortis sed. Quisque nisi turpis, volutpat eu molestie nec, volutpat nec quam.

Quisque consectetur a mauris sit amet euismod. Vestibulum eu dui in lorem congue pulvinar. Cras accumsan, enim ac viverra interdum, eros nisi posuere diam, vitae eleifend magna nibh ac mauris. Ut laoreet volutpat mi nec ullamcorper. Aenean dignissim egestas rutrum. Pellentesque aliquet fermentum arcu at pharetra. Sed et facilisis lorem, at varius felis. Suspendisse ac semper elit.

Nulla tempor hendrerit sem. Mauris posuere pellentesque tortor, sit amet fringilla purus facilisis sit amet. Vivamus consectetur ut lorem ac varius. Nulla consectetur tempor lorem, eget semper enim vestibulum quis. Donec convallis auctor diam, vitae rutrum nibh. Sed vitae odio fringilla, egestas mi at, interdum diam. Donec sodales sem quis turpis fermentum aliquam. In hac habitasse platea dictumst. Suspendisse in ligula consectetur, varius magna in, malesuada mauris. Vestibulum sapien arcu, ullamcorper eget ipsum ac, volutpat convallis nisi. Cras gravida porta nunc a sagittis. Nullam feugiat condimentum ante, eget rhoncus lacus.

Phasellus sit amet condimentum nisl. Aliquam erat volutpat. Sed egestas pretium dolor. Nam est tellus, tristique vel risus a, pharetra dictum urna. Maecenas ultrices elementum semper. Etiam posuere nibh non enim malesuada finibus. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Donec egestas elit eget magna molestie sodales. Maecenas dictum eros id dolor aliquam faucibus. Nullam vehicula ligula sed congue viverra. Curabitur sed purus luctus, pharetra risus quis, gravida nibh.

In in ultrices tellus. Sed vel posuere quam. Curabitur rutrum mi dignissim odio elementum semper sed vel ex. Quisque nec mi nulla. In purus metus, laoreet non nisl eu, mattis venenatis risus. Vestibulum quis lacinia lorem, maximus volutpat turpis. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Praesent nec molestie quam. Sed condimentum justo quis erat placerat, ac hendrerit urna ullamcorper. Donec venenatis tempor tempus. Nunc rhoncus venenatis nibh, nec condimentum est euismod vel. Proin nec sapien tincidunt, bibendum sapien molestie, aliquam nisl.

Aliquam sed quam massa. Nulla commodo eleifend est non accumsan. Praesent ac purus eget lectus dapibus fringilla. Ut sollicitudin tristique ex, vitae fermentum sapien. Maecenas felis risus, consequat a laoreet nec, commodo non nulla. Proin ornare lacus sit amet ante laoreet fringilla. Mauris molestie non massa ut condimentum. Morbi feugiat, enim vel consequat faucibus, nisi massa posuere dolor, id scelerisque urna leo vel erat. Pellentesque magna tellus, placerat ut cursus ornare, lacinia at eros. In id tellus pellentesque, rutrum lorem vitae, interdum urna. In consectetur et enim quis porta. Phasellus ultrices, nulla non placerat auctor, ex eros molestie erat, in blandit velit orci eget nisl. Cras at tellus eu neque tempor gravida nec at mauris.

Quisque faucibus venenatis dignissim. Aliquam erat volutpat. Mauris sit amet diam a sem maximus auctor. Nulla non libero a orci convallis consectetur. Integer auctor, dolor nec dictum scelerisque, arcu lacus sagittis mi, sed malesuada nisl lorem eu diam. Curabitur feugiat dui quis porttitor mattis. Nulla nec fringilla ante. Curabitur ac massa elementum, congue est eu, p
`

func getLoremIpsum() string {
	paragraphs := strings.Split(loremIpsum, "\n\n")

	paragraphsStart := rand.Intn(len(paragraphs))
	paragraphsEnd := paragraphsStart + rand.Intn(len(paragraphs)-paragraphsStart)

	return strings.Join(paragraphs[paragraphsStart:paragraphsEnd], "\n\n")
}

func seedJouranls() error {
	var user *models.User
	var journalTypes []*models.JournalType
	var ratings []*models.Rating

	err := db.First(&user).Error
	if err != nil {
		return err
	}

	err = db.Find(&journalTypes).Error
	if err != nil {
		return err
	}

	err = db.Find(&ratings).Error
	if err != nil {
		return err
	}

	var base = models.Base{
		CreatorID:     user.ID,
		LastUpdaterID: user.ID,
	}

	journals := []*models.Journal{}

	for i := 0; i < 45; i++ {
		entriesOnDay := rand.Intn(5)
		for j := 0; j < entriesOnDay; j++ {
			date := time.Now().AddDate(0, 0, -i)
			base.CreatedAt = date
			base.UpdatedAt = date

			journalType := journalTypes[rand.Intn(len(journalTypes))]
			rating := ratings[rand.Intn(len(ratings))]

			journals = append(journals, &models.Journal{
				Base:        &base,
				Date:        &date,
				Entry:       getLoremIpsum(),
				JournalType: journalType,
				Rating:      rating,
			})
		}
	}

	err = db.Create(&journals).Error
	if err != nil {
		return err
	}

	return nil
}
